<script lang="ts">
  import { Download, LoaderCircle } from '@lucide/svelte';
  import { api } from '$lib/api';

  // "Export" for the jobs list: collects EVERY job the current filters match (not just
  // the page on screen) and downloads them as an .xlsx with Application Link, Title and
  // Company.
  //
  // `getParams` is read at click time, so the export always follows the filters in force
  // right now — including the fixed `scope` a company page pins. The profile-match
  // slider is deliberately not part of it: that is a client-side trim of the visible page
  // (see JobsView), not a search facet, so there is nothing to send.
  //
  // The spreadsheet code is imported on the click, not at load: /jobs is the busiest
  // page on the site and almost nobody exports.
  let {
    getParams,
    total,
    disabled = false,
  }: {
    getParams: () => URLSearchParams;
    total: number;
    disabled?: boolean;
  } = $props();

  let status = $state<'idle' | 'running' | 'error'>('idle');
  let done = $state(0);
  let target = $state(0);
  let controller: AbortController | null = null;

  // Leaving the page mid-export cancels the remaining requests instead of letting them
  // finish into a download nobody is waiting for.
  $effect(() => () => controller?.abort());

  const capped = $derived(total > 10000);
  const label = $derived(
    status === 'running'
      ? target > 0
        ? `Exporting ${done.toLocaleString()} / ${target.toLocaleString()}`
        : 'Exporting…'
      : status === 'error'
        ? 'Export failed — retry'
        : 'Export',
  );
  const hint = $derived(
    capped
      ? 'Download the first 10,000 matching jobs as a spreadsheet (Application Link, Title, Company)'
      : 'Download every matching job as a spreadsheet (Application Link, Title, Company)',
  );

  async function run() {
    if (status === 'running') return;
    status = 'running';
    done = 0;
    target = 0;
    controller = new AbortController();
    try {
      const { collectJobs, toExportRow, buildXlsx, exportFilename, XLSX_MIME } = await import(
        '$lib/jobsExport'
      );
      const params = getParams();
      const jobs = await collectJobs((limit, offset, signal) => api.searchJobs(params, limit, offset, signal), {
        signal: controller.signal,
        onProgress: (d, t) => {
          done = d;
          target = t;
        },
      });
      const bytes = buildXlsx(jobs.map(toExportRow));
      const url = URL.createObjectURL(new Blob([bytes], { type: XLSX_MIME }));
      const a = document.createElement('a');
      a.href = url;
      a.download = exportFilename();
      document.body.append(a);
      a.click();
      a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
      status = 'idle';
    } catch (err) {
      // An abort is the visitor leaving, not a failure worth showing.
      status = err instanceof DOMException && err.name === 'AbortError' ? 'idle' : 'error';
    } finally {
      controller = null;
    }
  }
</script>

<button
  type="button"
  onclick={run}
  disabled={disabled || total === 0 || status === 'running'}
  aria-label="Export jobs to spreadsheet"
  title={hint}
  class="inline-flex shrink-0 items-center gap-1.5 rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm font-medium text-foreground transition-colors hover:bg-accent focus-visible:border-ring focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 md:py-1 dark:bg-input/30"
>
  {#if status === 'running'}
    <LoaderCircle class="size-4 shrink-0 animate-spin" aria-hidden="true" />
  {:else}
    <Download class="size-4 shrink-0" aria-hidden="true" />
  {/if}
  <span class={status === 'idle' ? 'hidden sm:inline' : ''} aria-live="polite">{label}</span>
</button>
