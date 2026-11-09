import { createRoot, hydrateRoot } from "react-dom/client";

import { App } from "./app";
import "./site.css";

const root = document.getElementById("root");

if (root?.firstElementChild) {
  hydrateRoot(root, <App />);
} else if (root) {
  // The dev server serves the template without the prerendered page.
  createRoot(root).render(<App />);
}
