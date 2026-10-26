package koiapi

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiapi/middlewares"
	apiresponse "github.com/koihq/koi/pkg/koiapi/response"
	"github.com/koihq/koi/pkg/koidomain"
)

type CredentialsReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthRes struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Organization OrgRes `json:"organization"`
}

type MeRes struct {
	AuthRes
	ActiveBrand *BrandRes `json:"activeBrand"`
}

type OrgRes struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

func AuthToRes(me *koidomain.Me) AuthRes {
	return AuthRes{
		ID:    me.User.ID.String(),
		Email: me.User.Email,
		Organization: OrgRes{
			ID:   me.Organization.ID.String(),
			Name: me.Organization.Name,
			Role: me.Role,
		},
	}
}

type AuthHandler struct {
	querier   koidomain.AuthQuerier
	commander koidomain.AuthCommander
	sess      *scs.SessionManager
	auth      func(http.Handler) http.Handler
}

func NewAuthHandler(q koidomain.AuthQuerier, c koidomain.AuthCommander, sess *scs.SessionManager, auth func(http.Handler) http.Handler) *AuthHandler {
	return &AuthHandler{querier: q, commander: c, sess: sess, auth: auth}
}

func (h *AuthHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.With(middlewares.DecodeBody[CredentialsReq]()).Post("/signup", h.Signup)
		r.With(middlewares.DecodeBody[CredentialsReq]()).Post("/login", h.Login)
		r.Post("/logout", h.Logout)
		r.With(h.auth).Get("/me", h.Me)
	}
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	req := middlewares.Body[CredentialsReq](r)

	me, err := h.commander.Signup(r.Context(), koidomain.SignupParams{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	if err := h.saveUser(r, me); err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	apiresponse.JSON(w, r, http.StatusCreated, AuthToRes(me))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	req := middlewares.Body[CredentialsReq](r)
	me, err := h.commander.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	if err := h.saveUser(r, me); err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	apiresponse.JSON(w, r, http.StatusOK, AuthToRes(me))
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.sess.Destroy(r.Context()); err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	_, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	preferredBrandID, _ := uuid.Parse(h.sess.GetString(r.Context(), "active_brand_id"))
	me, err := h.querier.Me(r.Context(), preferredBrandID)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	res := MeRes{AuthRes: AuthToRes(me)}
	if me.ActiveBrand != nil {
		brand := BrandToResponse(me.ActiveBrand)
		res.ActiveBrand = &brand
	}

	apiresponse.JSON(w, r, http.StatusOK, res)
}

func (h *AuthHandler) saveUser(r *http.Request, me *koidomain.Me) error {
	if err := h.sess.RenewToken(r.Context()); err != nil {
		return err
	}

	h.sess.Put(r.Context(), "user_id", me.User.ID.String())
	h.sess.Remove(r.Context(), "active_brand_id")

	return nil
}
