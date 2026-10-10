<script lang="ts">
  import { Check } from '@lucide/svelte';
  // The same posting once more, now inside a mocked search-result row rather than a
  // bare card — this is the shape it actually reaches you in, and the checkmark is the
  // one new fact this stage adds: it's live and searchable, not just correct.
</script>

<div class="flex h-28 flex-col justify-center gap-2" aria-hidden="true">
  <div class="flex items-center gap-3 rounded-md border border-border bg-card px-3 py-2.5">
    <div class="size-7 shrink-0 rounded bg-secondary"></div>
    <div class="min-w-0 flex-1">
      <p class="truncate text-xs font-medium">Senior Software Engineer</p>
      <p class="truncate text-[11px] text-muted-foreground">Berlin, DE · Remote-friendly</p>
    </div>
    <Check class="size-3.5 shrink-0 text-brand" />
  </div>
  <div class="pl-1 font-mono text-[11px] text-muted-foreground">indexed, searchable</div>
</div>
