/**
 * Characterization tests for the fork-specific "story" sticky param in
 * router.svelte.ts.
 *
 * The diff added "story" to STICKY_PARAMS alongside "desktop".
 *
 * The existing router.test.ts already covers:
 *   - "preserves story param in replaceParams" (one scenario)
 *   - All desktop sticky behavior
 *
 * This file covers the gaps: story sticky behavior across navigate,
 * navigateToSession, navigateFromSession, popstate, and param updates —
 * mirroring the desktop tests to prove story has exactly the same behavior.
 */

import {
  describe,
  it,
  expect,
  vi,
  afterEach,
} from "vitest";
import { RouterStore } from "./router.svelte.js";

function setURL(path: string) {
  window.history.replaceState(null, "", path);
}

describe("RouterStore story sticky param — navigate", () => {
  let store: RouterStore;

  afterEach(() => {
    store?.destroy();
    setURL("/");
  });

  it("preserves story param across navigate to another route", () => {
    setURL("/usage?story=1");
    store = new RouterStore();
    store.navigate("sessions");
    expect(window.location.search).toContain("story=1");
    expect(store.params.story).toBe("1");
  });

  it("preserves story param across two consecutive navigations", () => {
    setURL("/sessions?story=abc");
    store = new RouterStore();
    store.navigate("insights");
    expect(window.location.search).toContain("story=abc");
    store.navigate("pinned");
    expect(window.location.search).toContain("story=abc");
  });

  it("non-sticky param is dropped by navigate (story is NOT dropped)", () => {
    setURL("/sessions?story=1&custom_param=hello");
    store = new RouterStore();
    store.navigate("insights");
    // story is sticky — preserved
    expect(window.location.search).toContain("story=1");
    // custom_param is NOT sticky — dropped
    expect(window.location.search).not.toContain("custom_param");
  });

  it("routing params with story override the sticky story value", () => {
    setURL("/sessions?story=old");
    store = new RouterStore();
    store.navigate("usage", { story: "new" });
    expect(window.location.search).toContain("story=new");
    expect(store.params.story).toBe("new");
  });

  it("updated story value persists in subsequent navigation", () => {
    setURL("/sessions?story=v1");
    store = new RouterStore();
    store.navigate("usage", { story: "v2" });
    store.navigate("trends");
    expect(window.location.search).toContain("story=v2");
  });

  it("navigate includes story in params record", () => {
    setURL("/sessions?story=xyz");
    store = new RouterStore();
    store.navigate("insights");
    expect(store.params).toEqual({ story: "xyz" });
  });

  it("story and desktop can coexist as simultaneous sticky params", () => {
    setURL("/sessions?story=1&desktop");
    store = new RouterStore();
    store.navigate("insights");
    expect(window.location.search).toContain("story=1");
    expect(window.location.search).toContain("desktop=");
    expect(store.params.story).toBe("1");
    expect(store.params.desktop).toBe("");
  });
});

describe("RouterStore story sticky param — navigateToSession", () => {
  let store: RouterStore;

  afterEach(() => {
    store?.destroy();
    setURL("/");
  });

  it("preserves story param in navigateToSession", () => {
    setURL("/usage?story=1");
    store = new RouterStore();
    store.navigateToSession("abc-123");
    expect(window.location.pathname).toBe("/sessions/abc-123");
    expect(window.location.search).toContain("story=1");
    expect(store.params.story).toBe("1");
  });

  it("preserves story value through navigateToSession with extra params", () => {
    setURL("/usage?story=s1");
    store = new RouterStore();
    store.navigateToSession("abc-123", { msg: "last" });
    expect(window.location.search).toContain("story=s1");
    expect(window.location.search).toContain("msg=last");
  });
});

describe("RouterStore story sticky param — navigateFromSession", () => {
  let store: RouterStore;

  afterEach(() => {
    store?.destroy();
    setURL("/");
  });

  it("preserves story param in navigateFromSession", () => {
    setURL("/sessions/abc-123?story=1");
    store = new RouterStore();
    store.navigateFromSession();
    expect(window.location.pathname).toBe("/sessions");
    expect(window.location.search).toContain("story=1");
    expect(store.params.story).toBe("1");
  });

  it("preserves story alongside explicit filter params in navigateFromSession", () => {
    setURL("/sessions/abc-123?story=s2");
    store = new RouterStore();
    store.navigateFromSession({ project: "myproj" });
    expect(window.location.search).toContain("story=s2");
    expect(window.location.search).toContain("project=myproj");
    expect(store.params).toEqual({ story: "s2", project: "myproj" });
  });
});

describe("RouterStore story sticky param — replaceParams", () => {
  let store: RouterStore;

  afterEach(() => {
    store?.destroy();
    setURL("/");
  });

  it("preserves story param in replaceParams", () => {
    setURL("/usage?story=1");
    store = new RouterStore();
    store.replaceParams({ window_days: "7" });
    expect(window.location.search).toContain("story=1");
    expect(window.location.search).toContain("window_days=7");
    expect(store.params).toEqual({ story: "1", window_days: "7" });
  });

  it("non-sticky param is dropped by replaceParams but story is not", () => {
    setURL("/usage?story=1&transient=yes");
    store = new RouterStore();
    store.replaceParams({ other: "val" });
    expect(window.location.search).toContain("story=1");
    expect(window.location.search).not.toContain("transient=yes");
  });
});

describe("RouterStore story sticky param — popstate", () => {
  let store: RouterStore;

  afterEach(() => {
    store?.destroy();
    setURL("/");
  });

  it("refreshes story param value on popstate", () => {
    setURL("/sessions?story=v1");
    store = new RouterStore();
    setURL("/insights?story=v2");
    window.dispatchEvent(new PopStateEvent("popstate"));
    store.navigate("pinned");
    expect(window.location.search).toContain("story=v2");
  });

  it("removes story param on popstate to URL without it", () => {
    setURL("/sessions?story=1");
    store = new RouterStore();
    setURL("/insights");
    window.dispatchEvent(new PopStateEvent("popstate"));
    store.navigate("pinned");
    expect(window.location.search).not.toContain("story");
  });
});

describe("RouterStore story sticky param — buildSessionHref", () => {
  let store: RouterStore;

  afterEach(() => {
    store?.destroy();
    setURL("/");
  });

  it("buildSessionHref includes story param", () => {
    setURL("/usage?story=1");
    store = new RouterStore();
    const href = store.buildSessionHref("abc-123");
    expect(href).toContain("story=1");
    expect(href).toContain("/sessions/abc-123");
  });

  it("buildSessionHref works without story param", () => {
    setURL("/sessions");
    store = new RouterStore();
    const href = store.buildSessionHref("abc-123");
    expect(href).toBe("/sessions/abc-123");
  });
});

describe("RouterStore — non-sticky params are not preserved", () => {
  let store: RouterStore;

  afterEach(() => {
    store?.destroy();
    setURL("/");
  });

  it("a param that is NOT in STICKY_PARAMS is dropped across navigate", () => {
    // "window_days" is a routing param, not a sticky param
    setURL("/usage?window_days=7");
    store = new RouterStore();
    store.navigate("insights");
    expect(window.location.search).not.toContain("window_days");
  });

  it("a param that is NOT in STICKY_PARAMS is dropped across navigateToSession", () => {
    setURL("/usage?window_days=7");
    store = new RouterStore();
    store.navigateToSession("abc-123");
    expect(window.location.search).not.toContain("window_days");
  });

  it("proving story IS sticky while window_days IS NOT from the same URL", () => {
    setURL("/usage?story=1&window_days=7");
    store = new RouterStore();
    store.navigate("sessions");
    expect(window.location.search).toContain("story=1");
    expect(window.location.search).not.toContain("window_days");
  });
});
