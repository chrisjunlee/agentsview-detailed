<script lang="ts">
  import { sync } from "../../stores/sync.svelte.js";
  import {
    DATE_RANGE_PRESETS,
    isPresetActive,
    presetRange,
  } from "./dateRangeSelector.js";

  interface Props {
    from: string;
    to: string;
    fromTime?: string;
    toTime?: string;
    onChange: (from: string, to: string) => void;
    onTimeChange?: (fromTime: string, toTime: string) => void;
    onPreset?: (days: number) => void;
  }

  let {
    from,
    to,
    fromTime = "00:00",
    toTime = "00:00",
    onChange,
    onTimeChange,
    onPreset,
  }: Props = $props();

  const earliestSession = $derived(sync.stats?.earliest_session ?? null);

  function applyPreset(days: number) {
    if (days > 0 && onPreset) {
      onPreset(days);
      return;
    }
    const range = presetRange(days, earliestSession);
    onChange(range.from, range.to);
  }

  function handleFromChange(
    e: Event & { currentTarget: HTMLInputElement },
  ) {
    const val = e.currentTarget.value;
    if (val) onChange(val, to);
  }

  function handleToChange(
    e: Event & { currentTarget: HTMLInputElement },
  ) {
    const val = e.currentTarget.value;
    if (val) onChange(from, val);
  }

  let timeDebounce: ReturnType<typeof setTimeout>;

  function handleFromTimeChange(
    e: Event & { currentTarget: HTMLInputElement },
  ) {
    const val = e.currentTarget.value;
    clearTimeout(timeDebounce);
    timeDebounce = setTimeout(() => onTimeChange?.(val, toTime), 400);
  }

  function handleToTimeChange(
    e: Event & { currentTarget: HTMLInputElement },
  ) {
    const val = e.currentTarget.value;
    clearTimeout(timeDebounce);
    timeDebounce = setTimeout(() => onTimeChange?.(fromTime, val), 400);
  }
</script>

<div class="date-range-picker">
  <div class="presets">
    {#each DATE_RANGE_PRESETS as preset}
      <button
        class="preset-btn"
        class:active={isPresetActive(
          from,
          to,
          preset.days,
          earliestSession,
        )}
        onclick={() => applyPreset(preset.days)}
      >
        {preset.label}
      </button>
    {/each}
  </div>

  <div class="date-inputs">
    <input
      type="date"
      class="date-input"
      value={from}
      onchange={handleFromChange}
    />
    <input
      type="time"
      class="date-input time-input"
      value={fromTime}
      onchange={handleFromTimeChange}
      oninput={handleFromTimeChange}
    />
    <span class="date-sep">-</span>
    <input
      type="date"
      class="date-input"
      value={to}
      onchange={handleToChange}
    />
    <input
      type="time"
      class="date-input time-input"
      value={toTime}
      onchange={handleToTimeChange}
      oninput={handleToTimeChange}
    />
  </div>
</div>

<style>
  .date-range-picker {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .presets {
    display: flex;
    gap: 2px;
  }

  .preset-btn {
    height: 24px;
    padding: 0 8px;
    border-radius: var(--radius-sm);
    font-size: 11px;
    font-weight: 500;
    color: var(--text-muted);
    cursor: pointer;
    transition: background 0.1s, color 0.1s;
  }

  .preset-btn:hover {
    background: var(--bg-surface-hover);
    color: var(--text-secondary);
  }

  .preset-btn.active {
    background: var(--accent-blue);
    color: #fff;
  }

  .date-inputs {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .date-input {
    height: 24px;
    padding: 0 6px;
    background: var(--bg-inset);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-sm);
    font-size: 11px;
    color: var(--text-secondary);
    font-family: var(--font-mono);
  }

  .date-input:focus {
    outline: none;
    border-color: var(--accent-blue);
  }

  .time-input {
    width: 56px;
    padding: 0 2px;
    font-size: 10px;
  }

  .date-sep {
    color: var(--text-muted);
    font-size: 11px;
  }
</style>
