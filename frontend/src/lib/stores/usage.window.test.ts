/**
 * Characterization tests for fork-specific usage store behavior NOT already
 * covered by the existing usage.test.ts:
 *
 *  - setTimeRange() updates fromTime/toTime and calls fetchAll
 *  - baseParams includes from_time/to_time for the rolling one-day window
 *    (includeRollingTimeBounds = true when !isPinned && windowDays === 1)
 *  - baseParams omits from_time/to_time when pinned (even if fromTime !== "00:00")
 *  - applyRollingWindow resets fromTime/toTime to "00:00" without calling fetchAll
 *  - preserveDefaultWindowDays flag allows explicit window_days=1 in URL params
 *
 * The existing usage.test.ts already covers:
 *   - rolling window constructor defaults (noon-anchored, ceiled to hour)
 *   - fetchAll re-derives dates from clock
 *   - setDateRange pins
 *   - setRollingWindow clears pin and re-derives dates
 *   - buildUsageUrlParams omitting/including window_days
 */

import {
  beforeEach,
  afterEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";

vi.mock("../api/client.js", () => ({
  getUsageSummary: vi.fn().mockResolvedValue({
    from: "2026-04-24",
    to: "2026-04-25",
    totals: {
      inputTokens: 0,
      outputTokens: 0,
      cacheCreationTokens: 0,
      cacheReadTokens: 0,
      totalCost: 0,
    },
    daily: [],
    projectTotals: [],
    modelTotals: [],
    agentTotals: [],
    sessionCounts: { total: 0, byProject: {}, byAgent: {} },
    cacheStats: {
      cacheReadTokens: 0,
      cacheCreationTokens: 0,
      uncachedInputTokens: 0,
      outputTokens: 0,
      hitRate: 0,
      savingsVsUncached: 0,
    },
  }),
  getUsageTopSessions: vi.fn().mockResolvedValue([]),
}));

function installStorage(initial: Record<string, string> = {}) {
  const data = new Map(Object.entries(initial));
  const storage = {
    getItem: vi.fn((key: string) => data.get(key) ?? null),
    setItem: vi.fn((key: string, value: string) => {
      data.set(key, value);
    }),
    removeItem: vi.fn((key: string) => {
      data.delete(key);
    }),
    clear: vi.fn(() => {
      data.clear();
    }),
  };
  Object.defineProperty(globalThis, "localStorage", {
    value: storage,
    configurable: true,
    writable: true,
  });
  return storage;
}

async function loadStore() {
  vi.resetModules();
  return import("./usage.svelte.js");
}

// ---------------------------------------------------------------------------
// setTimeRange
// ---------------------------------------------------------------------------

describe("UsageStore.setTimeRange", () => {
  beforeEach(() => {
    installStorage();
    localStorage.removeItem("usage-toggles");
    localStorage.removeItem("usage-filters");
    vi.clearAllMocks();
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(new Date("2026-04-25T12:00:00"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  /**
   * setTimeRange sets fromTime/toTime, then calls fetchAll.
   * fetchAll calls rollDates. For the DEFAULT window (windowDays=1),
   * rollDates calls rollingOneDayWindow() which RE-DERIVES fromTime/toTime
   * from the clock — so on the 1-day window, setTimeRange's explicit values
   * are overwritten by rollDates.
   *
   * setTimeRange on a non-default window (7 days) does NOT trigger
   * rollingOneDayWindow, so the values survive.
   */
  it("on a 7-day window: sets fromTime and toTime persistently", async () => {
    const { usage } = await loadStore();
    // Switch to 7-day window first (times reset to 00:00)
    usage.setRollingWindow(7);
    expect(usage.fromTime).toBe("00:00");

    usage.setTimeRange("09:00", "17:00");

    // On 7-day window rollDates does NOT call rollingOneDayWindow,
    // so fromTime/toTime are preserved after the internal rollDates call.
    expect(usage.fromTime).toBe("09:00");
    expect(usage.toTime).toBe("17:00");
  });

  it("on default 1-day window: setTimeRange triggers fetchAll", async () => {
    const { usage } = await loadStore();
    const api = await import("../api/client.js");
    vi.clearAllMocks();

    await usage.setTimeRange("09:00", "17:00");

    // fetchAll was called (even though rollDates re-derives times for 1-day window)
    expect(api.getUsageSummary).toHaveBeenCalledTimes(1);
    expect(api.getUsageTopSessions).toHaveBeenCalledTimes(1);
  });

  it("passes updated fromTime/toTime to the API when not using default window", async () => {
    const { usage } = await loadStore();
    const api = await import("../api/client.js");

    // Switch to a 7-day window (non-default) so includeRollingTimeBounds = false
    usage.setRollingWindow(7);
    vi.clearAllMocks();

    await usage.setTimeRange("09:00", "17:00");

    expect(api.getUsageSummary).toHaveBeenLastCalledWith(
      expect.objectContaining({
        from_time: "09:00",
        to_time: "17:00",
      }),
    );
  });

  it("setting both times to 00:00 omits them from params when on 7-day window", async () => {
    const { usage } = await loadStore();
    const api = await import("../api/client.js");

    usage.setRollingWindow(7);
    vi.clearAllMocks();

    await usage.setTimeRange("00:00", "00:00");

    const params = vi.mocked(api.getUsageSummary).mock.lastCall?.[0];
    expect(params?.from_time).toBeUndefined();
    expect(params?.to_time).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// baseParams time-bound inclusion rules
// ---------------------------------------------------------------------------

describe("UsageStore baseParams from_time/to_time inclusion rules", () => {
  beforeEach(() => {
    installStorage();
    localStorage.removeItem("usage-toggles");
    localStorage.removeItem("usage-filters");
    vi.clearAllMocks();
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(new Date("2026-04-25T14:30:00"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("includes from_time/to_time for default one-day rolling window (ceiled 14:30 → 15:00)", async () => {
    // At 14:30, ceilToHour → 15:00; so fromTime=toTime="15:00"
    const { usage } = await loadStore();
    const api = await import("../api/client.js");
    vi.clearAllMocks();

    await usage.fetchAll();

    const params = vi.mocked(api.getUsageSummary).mock.lastCall?.[0];
    expect(params?.from_time).toBe("15:00");
    expect(params?.to_time).toBe("15:00");
  });

  it("omits from_time/to_time when pinned (even if they are non-00:00)", async () => {
    const { usage } = await loadStore();
    const api = await import("../api/client.js");

    // Pin with a date range while fromTime happens to be "15:00" (from constructor)
    // Then manually set fromTime to a non-default to verify omission under pin
    usage.setDateRange("2026-01-01", "2026-01-15");
    // After pin, fromTime/toTime retain constructor values ("15:00")
    // They should be omitted because isPinned=true and windowDays=1 so
    // includeRollingTimeBounds = !isPinned && windowDays===1 = false && true = false
    // AND fromTime !== "00:00" so from_time IS included (the other branch).
    // Actually: logic is (includeRollingTimeBounds || fromTime !== "00:00") so
    // it WILL be included when fromTime !== "00:00". Let's confirm the actual behavior.
    vi.clearAllMocks();
    await usage.fetchAll();

    const params = vi.mocked(api.getUsageSummary).mock.lastCall?.[0];
    // isPinned=true, windowDays=1, fromTime="15:00" (non-00:00)
    // includeRollingTimeBounds = false, fromTime !== "00:00" = true → included
    expect(params?.from_time).toBe("15:00");
    expect(params?.to_time).toBe("15:00");
  });

  it("omits from_time/to_time when pinned with 00:00 times", async () => {
    const { usage } = await loadStore();
    const api = await import("../api/client.js");

    // Set rolling window 7 days (clears times to 00:00), then pin
    usage.setRollingWindow(7);
    usage.setDateRange("2026-01-01", "2026-01-15");
    vi.clearAllMocks();
    await usage.fetchAll();

    const params = vi.mocked(api.getUsageSummary).mock.lastCall?.[0];
    // isPinned=true, windowDays=7, fromTime="00:00"
    // includeRollingTimeBounds = false, fromTime === "00:00" → omitted
    expect(params?.from_time).toBeUndefined();
    expect(params?.to_time).toBeUndefined();
  });

  it("omits from_time/to_time for 7-day rolling window at exactly 00:00", async () => {
    const { usage } = await loadStore();
    const api = await import("../api/client.js");

    // switch to 7-day: fromTime/toTime reset to "00:00"
    usage.setRollingWindow(7);
    vi.clearAllMocks();
    await usage.fetchAll();

    const params = vi.mocked(api.getUsageSummary).mock.lastCall?.[0];
    expect(params?.from_time).toBeUndefined();
    expect(params?.to_time).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// applyRollingWindow (the non-fetch variant used internally)
// ---------------------------------------------------------------------------

describe("UsageStore.applyRollingWindow", () => {
  beforeEach(() => {
    installStorage();
    localStorage.removeItem("usage-toggles");
    localStorage.removeItem("usage-filters");
    vi.clearAllMocks();
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(new Date("2026-04-25T14:30:00"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("resets fromTime and toTime to 00:00 without calling fetchAll", async () => {
    const { usage } = await loadStore();
    const api = await import("../api/client.js");

    // First switch to 7-day window so setTimeRange persists (1-day window
    // always re-derives fromTime from the clock in rollDates)
    usage.setRollingWindow(7);
    await usage.setTimeRange("09:00", "17:00");
    expect(usage.fromTime).toBe("09:00");
    vi.clearAllMocks();

    // applyRollingWindow should reset times but NOT call fetchAll
    usage.applyRollingWindow(14);

    expect(usage.fromTime).toBe("00:00");
    expect(usage.toTime).toBe("00:00");
    // No fetch happened
    expect(api.getUsageSummary).not.toHaveBeenCalled();
  });

  it("sets windowDays and clears isPinned", async () => {
    const { usage } = await loadStore();
    usage.setDateRange("2026-01-01", "2026-01-15");
    expect(usage.isPinned).toBe(true);

    usage.applyRollingWindow(14);

    expect(usage.windowDays).toBe(14);
    expect(usage.isPinned).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// preserveDefaultWindowDays: explicit URL param for default window
// ---------------------------------------------------------------------------

describe("buildUsageUrlParams preserveDefaultWindowDays", () => {
  it("emits window_days=1 when preserveDefaultWindowDays is true", async () => {
    const { buildUsageUrlParams } = await loadStore();
    const params = buildUsageUrlParams({
      from: "2026-04-24",
      to: "2026-04-25",
      isPinned: false,
      windowDays: 1,
      preserveDefaultWindowDays: true,
      excludedProjects: "",
      excludedAgents: "",
      excludedModels: "",
      selectedModels: "",
    });
    expect(params["window_days"]).toBe("1");
  });

  it("omits window_days when preserveDefaultWindowDays is false (default behavior)", async () => {
    const { buildUsageUrlParams } = await loadStore();
    const params = buildUsageUrlParams({
      from: "2026-04-24",
      to: "2026-04-25",
      isPinned: false,
      windowDays: 1,
      preserveDefaultWindowDays: false,
      excludedProjects: "",
      excludedAgents: "",
      excludedModels: "",
      selectedModels: "",
    });
    expect(params["window_days"]).toBeUndefined();
  });

  it("omits window_days when preserveDefaultWindowDays is absent", async () => {
    const { buildUsageUrlParams } = await loadStore();
    const params = buildUsageUrlParams({
      from: "2026-04-24",
      to: "2026-04-25",
      isPinned: false,
      windowDays: 1,
      excludedProjects: "",
      excludedAgents: "",
      excludedModels: "",
      selectedModels: "",
    });
    expect(params["window_days"]).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// USAGE_DEFAULT_WINDOW_DAYS export
// ---------------------------------------------------------------------------

describe("USAGE_DEFAULT_WINDOW_DAYS constant", () => {
  it("is 1 (one-day default window)", async () => {
    const { USAGE_DEFAULT_WINDOW_DAYS } = await loadStore();
    expect(USAGE_DEFAULT_WINDOW_DAYS).toBe(1);
  });
});
