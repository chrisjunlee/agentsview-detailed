/**
 * Characterization tests for fork-specific analytics store behavior NOT covered
 * by the existing analytics.test.ts:
 *
 *  - fromTime/toTime default to "00:00"
 *  - baseParams omits from_time/to_time when values are "00:00"
 *  - baseParams includes from_time/to_time ONLY when values !== "00:00"
 *  - setTimeRange(fromTime, toTime) sets both fields and triggers fetchAll
 *  - setRollingWindow resets fromTime/toTime back to "00:00"
 */

import {
  describe,
  it,
  expect,
  vi,
  beforeEach,
  afterEach,
} from "vitest";
import * as api from "../api/client.js";
import type {
  AnalyticsSummary,
  ActivityResponse,
  HeatmapResponse,
  ProjectsAnalyticsResponse,
  HourOfWeekResponse,
  SessionShapeResponse,
  VelocityResponse,
  ToolsAnalyticsResponse,
  TopSessionsResponse,
} from "../api/types.js";

vi.mock("../api/client.js", () => ({
  getAnalyticsSummary: vi.fn(),
  getAnalyticsActivity: vi.fn(),
  getAnalyticsHeatmap: vi.fn(),
  getAnalyticsProjects: vi.fn(),
  getAnalyticsHourOfWeek: vi.fn(),
  getAnalyticsSessionShape: vi.fn(),
  getAnalyticsVelocity: vi.fn(),
  getAnalyticsTools: vi.fn(),
  getAnalyticsTopSessions: vi.fn(),
  getAnalyticsSignals: vi.fn(),
}));

function makeSummary(): AnalyticsSummary {
  return {
    total_sessions: 0,
    total_messages: 0,
    total_output_tokens: 0,
    token_reporting_sessions: 0,
    active_projects: 0,
    active_days: 0,
    avg_messages: 0,
    median_messages: 0,
    p90_messages: 0,
    most_active_project: "",
    concentration: 0,
    agents: {},
  };
}

function makeActivity(): ActivityResponse {
  return { granularity: "day", series: [] };
}

function makeHeatmap(): HeatmapResponse {
  return {
    metric: "messages",
    entries: [],
    levels: { l1: 1, l2: 5, l3: 10, l4: 20 },
    entries_from: "2024-01-01",
  };
}

function makeProjects(): ProjectsAnalyticsResponse {
  return { projects: [] };
}

function makeHourOfWeek(): HourOfWeekResponse {
  return { cells: [] };
}

function makeSessionShape(): SessionShapeResponse {
  return {
    count: 0,
    length_distribution: [],
    duration_distribution: [],
    autonomy_distribution: [],
  };
}

function makeVelocity(): VelocityResponse {
  return {
    overall: {
      turn_cycle_sec: { p50: 0, p90: 0 },
      first_response_sec: { p50: 0, p90: 0 },
      msgs_per_active_min: 0,
      chars_per_active_min: 0,
      tool_calls_per_active_min: 0,
    },
    by_agent: [],
    by_complexity: [],
  };
}

function makeTools(): ToolsAnalyticsResponse {
  return {
    total_calls: 0,
    by_category: [],
    by_agent: [],
    trend: [],
  };
}

function makeTopSessions(): TopSessionsResponse {
  return { metric: "messages", sessions: [] };
}

function mockAllAPIs() {
  vi.mocked(api.getAnalyticsSummary).mockResolvedValue(makeSummary());
  vi.mocked(api.getAnalyticsActivity).mockResolvedValue(makeActivity());
  vi.mocked(api.getAnalyticsHeatmap).mockResolvedValue(makeHeatmap());
  vi.mocked(api.getAnalyticsProjects).mockResolvedValue(makeProjects());
  vi.mocked(api.getAnalyticsHourOfWeek).mockResolvedValue(makeHourOfWeek());
  vi.mocked(api.getAnalyticsSessionShape).mockResolvedValue(makeSessionShape());
  vi.mocked(api.getAnalyticsVelocity).mockResolvedValue(makeVelocity());
  vi.mocked(api.getAnalyticsTools).mockResolvedValue(makeTools());
  vi.mocked(api.getAnalyticsTopSessions).mockResolvedValue(makeTopSessions());
  vi.mocked(api.getAnalyticsSignals).mockResolvedValue({
    scored_sessions: 0,
    unscored_sessions: 0,
    grade_distribution: {},
    avg_health_score: null,
    outcome_distribution: {},
    outcome_confidence_distribution: {},
    tool_health: {
      total_failure_signals: 0,
      total_retries: 0,
      total_edit_churn: 0,
      sessions_with_failures: 0,
      failure_rate: 0,
    },
    context_health: {
      avg_compaction_count: 0,
      sessions_with_compaction: 0,
      mid_task_compaction_count: 0,
      sessions_with_mid_task_compaction: 0,
      sessions_with_context_data: 0,
      avg_context_pressure: null,
      high_pressure_sessions: 0,
    },
    trend: [],
    by_agent: [],
    by_project: [],
  });
}

async function loadAnalyticsStore() {
  vi.resetModules();
  vi.clearAllMocks();
  mockAllAPIs();
  return import("./analytics.svelte.js");
}

// ---------------------------------------------------------------------------
// 1. Default state
// ---------------------------------------------------------------------------

describe("AnalyticsStore fromTime/toTime defaults", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(new Date("2026-04-25T12:00:00"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('fromTime defaults to "00:00"', async () => {
    const { analytics } = await loadAnalyticsStore();
    expect(analytics.fromTime).toBe("00:00");
  });

  it('toTime defaults to "00:00"', async () => {
    const { analytics } = await loadAnalyticsStore();
    expect(analytics.toTime).toBe("00:00");
  });
});

// ---------------------------------------------------------------------------
// 2. baseParams omits from_time/to_time when "00:00"
// ---------------------------------------------------------------------------

describe("AnalyticsStore baseParams omits time params at default 00:00", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(new Date("2026-04-25T12:00:00"));
    vi.clearAllMocks();
    mockAllAPIs();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("fetchAll omits from_time from summary request when fromTime is 00:00", async () => {
    const { analytics } = await loadAnalyticsStore();

    await analytics.fetchAll();

    const params = vi.mocked(api.getAnalyticsSummary).mock.lastCall?.[0];
    expect(params).toBeDefined();
    expect(params?.from_time).toBeUndefined();
  });

  it("fetchAll omits to_time from summary request when toTime is 00:00", async () => {
    const { analytics } = await loadAnalyticsStore();

    await analytics.fetchAll();

    const params = vi.mocked(api.getAnalyticsSummary).mock.lastCall?.[0];
    expect(params?.to_time).toBeUndefined();
  });

  it("fetchAll omits from_time/to_time from activity request at default", async () => {
    const { analytics } = await loadAnalyticsStore();

    await analytics.fetchAll();

    const params = vi.mocked(api.getAnalyticsActivity).mock.lastCall?.[0];
    expect(params?.from_time).toBeUndefined();
    expect(params?.to_time).toBeUndefined();
  });

  it("fetchAll omits from_time/to_time from heatmap request at default", async () => {
    const { analytics } = await loadAnalyticsStore();

    await analytics.fetchAll();

    const params = vi.mocked(api.getAnalyticsHeatmap).mock.lastCall?.[0];
    expect(params?.from_time).toBeUndefined();
    expect(params?.to_time).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// 3. baseParams INCLUDES from_time/to_time when not "00:00"
// ---------------------------------------------------------------------------

describe("AnalyticsStore baseParams includes time params when non-default", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(new Date("2026-04-25T12:00:00"));
    vi.clearAllMocks();
    mockAllAPIs();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("setTimeRange causes summary request to include from_time", async () => {
    const { analytics } = await loadAnalyticsStore();
    vi.clearAllMocks();
    mockAllAPIs();

    await analytics.setTimeRange("09:00", "17:00");

    const params = vi.mocked(api.getAnalyticsSummary).mock.lastCall?.[0];
    expect(params?.from_time).toBe("09:00");
  });

  it("setTimeRange causes summary request to include to_time", async () => {
    const { analytics } = await loadAnalyticsStore();
    vi.clearAllMocks();
    mockAllAPIs();

    await analytics.setTimeRange("09:00", "17:00");

    const params = vi.mocked(api.getAnalyticsSummary).mock.lastCall?.[0];
    expect(params?.to_time).toBe("17:00");
  });

  it("setTimeRange causes activity request to include from_time/to_time", async () => {
    const { analytics } = await loadAnalyticsStore();
    vi.clearAllMocks();
    mockAllAPIs();

    await analytics.setTimeRange("08:00", "20:00");

    const params = vi.mocked(api.getAnalyticsActivity).mock.lastCall?.[0];
    expect(params?.from_time).toBe("08:00");
    expect(params?.to_time).toBe("20:00");
  });

  it("setTimeRange causes heatmap request to include from_time/to_time", async () => {
    const { analytics } = await loadAnalyticsStore();
    vi.clearAllMocks();
    mockAllAPIs();

    await analytics.setTimeRange("06:00", "22:00");

    const params = vi.mocked(api.getAnalyticsHeatmap).mock.lastCall?.[0];
    expect(params?.from_time).toBe("06:00");
    expect(params?.to_time).toBe("22:00");
  });

  it("asymmetric times: only from_time non-default includes only from_time", async () => {
    const { analytics } = await loadAnalyticsStore();
    vi.clearAllMocks();
    mockAllAPIs();

    await analytics.setTimeRange("09:00", "00:00");

    const params = vi.mocked(api.getAnalyticsSummary).mock.lastCall?.[0];
    expect(params?.from_time).toBe("09:00");
    expect(params?.to_time).toBeUndefined();
  });

  it("asymmetric times: only to_time non-default includes only to_time", async () => {
    const { analytics } = await loadAnalyticsStore();
    vi.clearAllMocks();
    mockAllAPIs();

    await analytics.setTimeRange("00:00", "17:00");

    const params = vi.mocked(api.getAnalyticsSummary).mock.lastCall?.[0];
    expect(params?.from_time).toBeUndefined();
    expect(params?.to_time).toBe("17:00");
  });
});

// ---------------------------------------------------------------------------
// 4. setTimeRange sets state and triggers fetchAll
// ---------------------------------------------------------------------------

describe("AnalyticsStore.setTimeRange", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(new Date("2026-04-25T12:00:00"));
    vi.clearAllMocks();
    mockAllAPIs();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("updates fromTime on the store", async () => {
    const { analytics } = await loadAnalyticsStore();
    analytics.setTimeRange("10:00", "18:00");
    expect(analytics.fromTime).toBe("10:00");
  });

  it("updates toTime on the store", async () => {
    const { analytics } = await loadAnalyticsStore();
    analytics.setTimeRange("10:00", "18:00");
    expect(analytics.toTime).toBe("18:00");
  });

  it("triggers fetchAll (summary is called)", async () => {
    const { analytics } = await loadAnalyticsStore();
    vi.clearAllMocks();
    mockAllAPIs();

    await analytics.setTimeRange("10:00", "18:00");

    expect(api.getAnalyticsSummary).toHaveBeenCalledTimes(1);
  });

  it("triggers fetchAll (activity is called)", async () => {
    const { analytics } = await loadAnalyticsStore();
    vi.clearAllMocks();
    mockAllAPIs();

    await analytics.setTimeRange("10:00", "18:00");

    expect(api.getAnalyticsActivity).toHaveBeenCalledTimes(1);
  });

  it("setTimeRange with both 00:00 triggers fetch but omits time params", async () => {
    const { analytics } = await loadAnalyticsStore();
    // First set non-default to make state dirty
    analytics.setTimeRange("09:00", "17:00");
    vi.clearAllMocks();
    mockAllAPIs();

    await analytics.setTimeRange("00:00", "00:00");

    expect(analytics.fromTime).toBe("00:00");
    expect(analytics.toTime).toBe("00:00");
    expect(api.getAnalyticsSummary).toHaveBeenCalledTimes(1);
    const params = vi.mocked(api.getAnalyticsSummary).mock.lastCall?.[0];
    expect(params?.from_time).toBeUndefined();
    expect(params?.to_time).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// 5. setRollingWindow resets fromTime/toTime to "00:00"
// ---------------------------------------------------------------------------

describe("AnalyticsStore.setRollingWindow resets time range", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(new Date("2026-04-25T12:00:00"));
    vi.clearAllMocks();
    mockAllAPIs();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('resets fromTime to "00:00" after setTimeRange', async () => {
    const { analytics } = await loadAnalyticsStore();
    analytics.setTimeRange("09:00", "17:00");
    expect(analytics.fromTime).toBe("09:00");

    vi.clearAllMocks();
    mockAllAPIs();
    analytics.setRollingWindow(30);

    expect(analytics.fromTime).toBe("00:00");
  });

  it('resets toTime to "00:00" after setTimeRange', async () => {
    const { analytics } = await loadAnalyticsStore();
    analytics.setTimeRange("09:00", "17:00");
    expect(analytics.toTime).toBe("17:00");

    vi.clearAllMocks();
    mockAllAPIs();
    analytics.setRollingWindow(30);

    expect(analytics.toTime).toBe("00:00");
  });

  it("triggers fetchAll after resetting times (summary called)", async () => {
    const { analytics } = await loadAnalyticsStore();
    analytics.setTimeRange("09:00", "17:00");
    vi.clearAllMocks();
    mockAllAPIs();

    await analytics.setRollingWindow(7);

    expect(api.getAnalyticsSummary).toHaveBeenCalledTimes(1);
  });

  it("subsequent fetchAll omits time params after setRollingWindow", async () => {
    const { analytics } = await loadAnalyticsStore();
    analytics.setTimeRange("09:00", "17:00");
    vi.clearAllMocks();
    mockAllAPIs();

    await analytics.setRollingWindow(7);
    vi.clearAllMocks();
    mockAllAPIs();

    await analytics.fetchAll();

    const params = vi.mocked(api.getAnalyticsSummary).mock.lastCall?.[0];
    expect(params?.from_time).toBeUndefined();
    expect(params?.to_time).toBeUndefined();
  });

  it("also clears selectedDate, selectedDow, selectedHour", async () => {
    const { analytics } = await loadAnalyticsStore();
    analytics.selectedDate = "2026-04-20";
    analytics.selectedDow = 2;
    analytics.selectedHour = 9;
    analytics.setTimeRange("09:00", "17:00");
    vi.clearAllMocks();
    mockAllAPIs();

    analytics.setRollingWindow(14);

    expect(analytics.selectedDate).toBeNull();
    expect(analytics.selectedDow).toBeNull();
    expect(analytics.selectedHour).toBeNull();
  });
});
