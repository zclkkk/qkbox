export type Route = "engine" | "profiles" | "platform" | "diagnostics";

const defaultRoute: Route = "engine";
const routes = new Set<Route>(["engine", "profiles", "platform", "diagnostics"]);

function routeFromHash(): Route {
  const raw = window.location.hash.replace(/^#/, "") as Route;
  return routes.has(raw) ? raw : defaultRoute;
}

class RouterState {
  current = $state<Route>(defaultRoute);

  start() {
    this.current = routeFromHash();
    const sync = () => {
      this.current = routeFromHash();
    };
    window.addEventListener("hashchange", sync);
    return () => window.removeEventListener("hashchange", sync);
  }

  navigate(route: Route) {
    if (this.current === route) {
      return;
    }
    window.location.hash = route;
    this.current = route;
  }
}

export const router = new RouterState();
