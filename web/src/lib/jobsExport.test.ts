import { describe, expect, it } from 'vitest';
import {
  EXPORT_HEADERS,
  buildSheetXml,
  buildXlsx,
  collectJobs,
  exportFilename,
  toExportRow,
  type ExportableJob,
} from './jobsExport';

const job = (i: number, over: Partial<ExportableJob> = {}): ExportableJob => ({
  public_slug: `s${i}`,
  url: `https://example.com/jobs/${i}`,
  title: `Title ${i}`,
  company: `Company ${i}`,
  ...over,
});

describe('toExportRow', () => {
  it('maps the three exported columns and trims them', () => {
    expect(toExportRow(job(1, { title: '  Dev  ' }))).toEqual({
      link: 'https://example.com/jobs/1',
      title: 'Dev',
      company: 'Company 1',
    });
  });
});

describe('buildSheetXml', () => {
  it('writes the header, one hyperlink per http(s) link, and escapes markup', () => {
    const { sheet, links } = buildSheetXml([
      { link: 'https://a.com/?x=1&y=2', title: 'A <b> & "c"', company: 'Co' },
      { link: 'not a url', title: '=1+1', company: '' },
    ]);
    for (const h of EXPORT_HEADERS) expect(sheet).toContain(`>${h}</t>`);
    expect(links).toEqual(['https://a.com/?x=1&y=2']);
    expect(sheet).toContain('A &lt;b&gt; &amp; &quot;c&quot;');
    // Text cells only — a title starting with "=" must never become a formula.
    expect(sheet).not.toContain('<f>');
    expect(sheet.match(/<hyperlink /g)).toHaveLength(1);
  });

  it('drops characters XML cannot carry', () => {
    const { sheet } = buildSheetXml([{ link: '', title: 'a\u0001b', company: '' }]);
    expect(sheet).toContain('>ab</t>');
  });
});

describe('buildXlsx', () => {
  it('produces a zip holding the workbook parts', () => {
    const bytes = buildXlsx([toExportRow(job(1))]);
    // Local file header signature "PK\x03\x04", end-of-central-directory "PK\x05\x06".
    expect([...bytes.slice(0, 4)]).toEqual([0x50, 0x4b, 0x03, 0x04]);
    const text = new TextDecoder().decode(bytes);
    for (const part of [
      '[Content_Types].xml',
      'xl/workbook.xml',
      'xl/worksheets/sheet1.xml',
      'xl/worksheets/_rels/sheet1.xml.rels',
    ]) {
      expect(text).toContain(part);
    }
    expect(text).toContain('PK\x05\x06');
  });
});

describe('collectJobs', () => {
  const all = Array.from({ length: 250 }, (_, i) => job(i));
  const serve = (extra?: (offset: number, items: ExportableJob[]) => void) =>
    async (limit: number, offset: number) => {
      const items = all.slice(offset, offset + limit);
      extra?.(offset, items);
      return { items, total: all.length };
    };

  it('walks every page and keeps result order', async () => {
    const got = await collectJobs(serve());
    expect(got.map((j) => j.public_slug)).toEqual(all.map((j) => j.public_slug));
  });

  it('drops a row that shifted into the next window', async () => {
    const got = await collectJobs(
      serve((offset, items) => {
        if (offset === 100) items.unshift(all[99]!);
      }),
    );
    expect(got).toHaveLength(250);
  });

  it('stops at the cap', async () => {
    const got = await collectJobs(serve(), { max: 120 });
    expect(got).toHaveLength(120);
  });

  it('returns nothing for an empty result', async () => {
    expect(await collectJobs(async () => ({ items: [], total: 0 }))).toEqual([]);
  });

  it('reports progress up to the target', async () => {
    const seen: number[] = [];
    await collectJobs(serve(), { onProgress: (n) => seen.push(n) });
    expect(seen.at(-1)).toBe(250);
  });
});

describe('exportFilename', () => {
  it('is dated', () => {
    expect(exportFilename(new Date('2026-10-10T12:00:00Z'))).toBe('freehire-jobs-2026-10-10.xlsx');
  });
});
