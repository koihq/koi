import { renderToString } from "react-dom/server";

import { pageFor, pages } from "./pages";

export { pages };

export function render(path: string) {
  return renderToString(pageFor(path).render());
}
