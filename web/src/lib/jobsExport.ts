// Export the jobs a filter set matches into a spreadsheet: Application Link, Title,
// Company — one row per posting.
//
// Pure by design — no Svelte, no DOM, no `$app/*` — so it unit-tests in plain Node.
// The click, the progress text and the file download live in ExportJobsButton.svelte.
//
// The workbook is written by hand rather than through a library: an .xlsx is a ZIP of
// a few XML parts, and this one needs a single sheet of three text columns. A
// dependency for that would be a lot of code shipped to every visitor of /jobs for a
// button most of them never press (the component imports this module on click only).

import { API_WINDOW } from './pagination';

/** The API refuses `limit` above this (internal/api/handler/handler.go, maxLimit). */
export const EXPORT_PAGE_SIZE = 100;

/** Most rows one export collects — what the search API will serve at all. */
export const EXPORT_MAX_ROWS = API_WINDOW;

/** How many pages are in flight at once. Enough to make 10k rows take seconds rather
 *  than minutes, few enough not to hammer the search backend. */
const CONCURRENCY = 4;

/** The fields an export row is built from — a subset of `Job`, so this module does not
 *  need the whole contract type and tests can pass plain objects. */
export interface ExportableJob {
  public_slug: string;
  url: string;
  title: string;
  company: string;
}

export interface ExportRow {
  link: string;
  title: string;
  company: string;
}

export const EXPORT_HEADERS = ['Application Link', 'Title', 'Company'] as const;

export function toExportRow(job: ExportableJob): ExportRow {
  return {
    link: (job.url ?? '').trim(),
    title: (job.title ?? '').trim(),
    company: (job.company ?? '').trim(),
  };
}

interface PageResult {
  items: ExportableJob[];
  total: number;
}

export interface CollectOptions {
  /** Called after each page lands with how many unique rows are held so far and the
   *  number we are aiming for. */
  onProgress?: (collected: number, target: number) => void;
  signal?: AbortSignal;
  max?: number;
}

/** Fetch every page of the current search, deduplicated by `public_slug`.
 *
 *  The first request tells us the total; the rest are then fetched a few at a time.
 *  Offset paging over a live index can shift a row into the next window between two
 *  requests, so the same posting may arrive twice — `public_slug` is the identity the
 *  feed itself dedups on (see Paginator's keyOf).
 *
 *  `target` is what we aim for (total, capped) — the search engine's total is an
 *  estimate, so the result may fall a little short of it. */
export async function collectJobs(
  fetchPage: (limit: number, offset: number, signal?: AbortSignal) => Promise<PageResult>,
  { onProgress, signal, max = EXPORT_MAX_ROWS }: CollectOptions = {},
): Promise<ExportableJob[]> {
  const seen = new Set<string>();
  const out: ExportableJob[] = [];
  const take = (items: ExportableJob[]) => {
    for (const job of items) {
      if (seen.has(job.public_slug)) continue;
      seen.add(job.public_slug);
      out.push(job);
    }
  };

  const first = await fetchPage(EXPORT_PAGE_SIZE, 0, signal);
  take(first.items);
  const target = Math.min(first.total, max);
  onProgress?.(out.length, target);

  const offsets: number[] = [];
  for (let offset = EXPORT_PAGE_SIZE; offset < target; offset += EXPORT_PAGE_SIZE) {
    offsets.push(offset);
  }

  // Pages land in whatever order they resolve, but the rows must come out in result
  // order — the sort the visitor chose is part of what they are exporting.
  const pages = new Map<number, ExportableJob[]>();
  let next = 0;
  const worker = async () => {
    while (next < offsets.length) {
      signal?.throwIfAborted();
      const offset = offsets[next++]!;
      const limit = Math.min(EXPORT_PAGE_SIZE, target - offset);
      const page = await fetchPage(limit, offset, signal);
      pages.set(offset, page.items);
      onProgress?.(
        Math.min(
          target,
          first.items.length + [...pages.values()].reduce((n, p) => n + p.length, 0),
        ),
        target,
      );
    }
  };
  await Promise.all(Array.from({ length: Math.min(CONCURRENCY, offsets.length) }, worker));

  for (const offset of offsets) take(pages.get(offset) ?? []);
  return out.slice(0, max);
}

/** `freehire-jobs-2026-10-10.xlsx` */
export function exportFilename(now = new Date()): string {
  return `freehire-jobs-${now.toISOString().slice(0, 10)}.xlsx`;
}

// ---------------------------------------------------------------------------
// XLSX writer
// ---------------------------------------------------------------------------

const XLSX_MIME = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';
export { XLSX_MIME };

// XML 1.0 forbids most control characters even escaped; a stray one in a scraped job
// title would make Excel report the whole file as corrupt.
// eslint-disable-next-line no-control-regex -- stripping exactly these is the point
const INVALID_XML = /[\u0000-\u0008\u000B\u000C\u000E-\u001F\uFFFE\uFFFF]/g;

function esc(s: string): string {
  return s
    .replace(INVALID_XML, '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

/** Column letter for a 0-based index (A..Z is all this sheet needs, kept general). */
function colName(i: number): string {
  let n = i;
  let s = '';
  do {
    s = String.fromCharCode(65 + (n % 26)) + s;
    n = Math.floor(n / 26) - 1;
  } while (n >= 0);
  return s;
}

/** Strings are written inline, so a cell is always text — a title like `=1+1` can
 *  never be evaluated as a formula by the spreadsheet. */
function textCell(ref: string, value: string, style?: number): string {
  const s = style ? ` s="${style}"` : '';
  // xml:space keeps leading/trailing spaces the title may carry.
  return `<c r="${ref}" t="inlineStr"${s}><is><t xml:space="preserve">${esc(value)}</t></is></c>`;
}

function isHttpUrl(s: string): boolean {
  return /^https?:\/\/\S+$/i.test(s);
}

export function buildSheetXml(rows: ExportRow[]): { sheet: string; links: string[] } {
  const links: string[] = [];
  const lines: string[] = [];

  lines.push(
    `<row r="1">${EXPORT_HEADERS.map((h, c) => textCell(`${colName(c)}1`, h, 1)).join('')}</row>`,
  );
  const hyperlinks: string[] = [];
  rows.forEach((row, i) => {
    const r = i + 2;
    const linkIsUrl = isHttpUrl(row.link);
    const cells = [
      textCell(`A${r}`, row.link, linkIsUrl ? 2 : undefined),
      textCell(`B${r}`, row.title),
      textCell(`C${r}`, row.company),
    ];
    lines.push(`<row r="${r}">${cells.join('')}</row>`);
    if (linkIsUrl) {
      links.push(row.link);
      hyperlinks.push(`<hyperlink ref="A${r}" r:id="rId${links.length}"/>`);
    }
  });

  const last = rows.length + 1;
  const sheet =
    `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
    `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
    `<dimension ref="A1:C${last}"/>` +
    `<sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews>` +
    `<sheetFormatPr defaultRowHeight="15"/>` +
    `<cols><col min="1" max="1" width="60" customWidth="1"/><col min="2" max="2" width="50" customWidth="1"/><col min="3" max="3" width="30" customWidth="1"/></cols>` +
    `<sheetData>${lines.join('')}</sheetData>` +
    `<autoFilter ref="A1:C${last}"/>` +
    (hyperlinks.length ? `<hyperlinks>${hyperlinks.join('')}</hyperlinks>` : '') +
    `</worksheet>`;
  return { sheet, links };
}

const STYLES_XML =
  `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
  `<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
  `<fonts count="3">` +
  `<font><sz val="11"/><name val="Calibri"/></font>` +
  `<font><b/><sz val="11"/><name val="Calibri"/></font>` +
  `<font><u/><sz val="11"/><color rgb="FF0563C1"/><name val="Calibri"/></font>` +
  `</fonts>` +
  `<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>` +
  `<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>` +
  `<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +
  `<cellXfs count="3">` +
  `<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>` +
  `<xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/>` +
  `<xf numFmtId="0" fontId="2" fillId="0" borderId="0" xfId="0" applyFont="1"/>` +
  `</cellXfs>` +
  `<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles>` +
  `</styleSheet>`;

const NS_REL = 'http://schemas.openxmlformats.org/package/2006/relationships';
const NS_DOC_REL = 'http://schemas.openxmlformats.org/officeDocument/2006/relationships';

/** The parts of the package, as [path, xml] pairs, in the order they are zipped. */
export function buildXlsxParts(rows: ExportRow[]): [string, string][] {
  const { sheet, links } = buildSheetXml(rows);
  const parts: [string, string][] = [
    [
      '[Content_Types].xml',
      `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
        `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
        `<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
        `<Default Extension="xml" ContentType="application/xml"/>` +
        `<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
        `<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
        `<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>` +
        `</Types>`,
    ],
    [
      '_rels/.rels',
      `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
        `<Relationships xmlns="${NS_REL}"><Relationship Id="rId1" Type="${NS_DOC_REL}/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
    ],
    [
      'xl/workbook.xml',
      `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
        `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="${NS_DOC_REL}">` +
        `<sheets><sheet name="Jobs" sheetId="1" r:id="rId1"/></sheets>` +
        `<definedNames><definedName name="_xlnm._FilterDatabase" localSheetId="0" hidden="1">Jobs!$A$1:$C$${rows.length + 1}</definedName></definedNames>` +
        `</workbook>`,
    ],
    [
      'xl/_rels/workbook.xml.rels',
      `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
        `<Relationships xmlns="${NS_REL}">` +
        `<Relationship Id="rId1" Type="${NS_DOC_REL}/worksheet" Target="worksheets/sheet1.xml"/>` +
        `<Relationship Id="rId2" Type="${NS_DOC_REL}/styles" Target="styles.xml"/>` +
        `</Relationships>`,
    ],
    ['xl/styles.xml', STYLES_XML],
    ['xl/worksheets/sheet1.xml', sheet],
  ];
  if (links.length) {
    parts.push([
      'xl/worksheets/_rels/sheet1.xml.rels',
      `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
        `<Relationships xmlns="${NS_REL}">` +
        links
          .map(
            (url, i) =>
              `<Relationship Id="rId${i + 1}" Type="${NS_DOC_REL}/hyperlink" Target="${esc(url)}" TargetMode="External"/>`,
          )
          .join('') +
        `</Relationships>`,
    ]);
  }
  return parts;
}

// --- minimal ZIP (store only) ----------------------------------------------

let crcTable: Uint32Array | null = null;
function crc32(data: Uint8Array): number {
  if (!crcTable) {
    crcTable = new Uint32Array(256);
    for (let n = 0; n < 256; n++) {
      let c = n;
      for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
      crcTable[n] = c >>> 0;
    }
  }
  let crc = 0xffffffff;
  for (let i = 0; i < data.length; i++) crc = crcTable[(crc ^ data[i]!) & 0xff]! ^ (crc >>> 8);
  return (crc ^ 0xffffffff) >>> 0;
}

/** A ZIP archive with every entry stored uncompressed. Excel opens these without
 *  complaint; the file is larger than a deflated one, which for three text columns is
 *  still well under a megabyte per thousand rows. */
export function zipStore(entries: [string, Uint8Array][]): Uint8Array<ArrayBuffer> {
  const enc = new TextEncoder();
  // 1980-01-01 00:00 — the format has no "unset", and a fixed stamp keeps output
  // reproducible.
  const DOS_TIME = 0;
  const DOS_DATE = (0 << 9) | (1 << 5) | 1;
  const UTF8_FLAG = 0x0800;

  const chunks: Uint8Array[] = [];
  const central: Uint8Array[] = [];
  let offset = 0;

  for (const [name, data] of entries) {
    const nameBytes = enc.encode(name);
    const crc = crc32(data);

    const local = new Uint8Array(30 + nameBytes.length);
    const lv = new DataView(local.buffer);
    lv.setUint32(0, 0x04034b50, true);
    lv.setUint16(4, 20, true); // version needed
    lv.setUint16(6, UTF8_FLAG, true);
    lv.setUint16(8, 0, true); // method: store
    lv.setUint16(10, DOS_TIME, true);
    lv.setUint16(12, DOS_DATE, true);
    lv.setUint32(14, crc, true);
    lv.setUint32(18, data.length, true);
    lv.setUint32(22, data.length, true);
    lv.setUint16(26, nameBytes.length, true);
    lv.setUint16(28, 0, true);
    local.set(nameBytes, 30);
    chunks.push(local, data);

    const cd = new Uint8Array(46 + nameBytes.length);
    const cv = new DataView(cd.buffer);
    cv.setUint32(0, 0x02014b50, true);
    cv.setUint16(4, 20, true); // version made by
    cv.setUint16(6, 20, true); // version needed
    cv.setUint16(8, UTF8_FLAG, true);
    cv.setUint16(10, 0, true);
    cv.setUint16(12, DOS_TIME, true);
    cv.setUint16(14, DOS_DATE, true);
    cv.setUint32(16, crc, true);
    cv.setUint32(20, data.length, true);
    cv.setUint32(24, data.length, true);
    cv.setUint16(28, nameBytes.length, true);
    cv.setUint32(42, offset, true);
    cd.set(nameBytes, 46);
    central.push(cd);

    offset += local.length + data.length;
  }

  const centralSize = central.reduce((n, c) => n + c.length, 0);
  const end = new Uint8Array(22);
  const ev = new DataView(end.buffer);
  ev.setUint32(0, 0x06054b50, true);
  ev.setUint16(8, entries.length, true);
  ev.setUint16(10, entries.length, true);
  ev.setUint32(12, centralSize, true);
  ev.setUint32(16, offset, true);

  const all = [...chunks, ...central, end];
  const out = new Uint8Array(all.reduce((n, c) => n + c.length, 0));
  let pos = 0;
  for (const c of all) {
    out.set(c, pos);
    pos += c.length;
  }
  return out;
}

/** The finished workbook, ready to wrap in a Blob. */
export function buildXlsx(rows: ExportRow[]): Uint8Array<ArrayBuffer> {
  const enc = new TextEncoder();
  return zipStore(buildXlsxParts(rows).map(([name, xml]) => [name, enc.encode(xml)]));
}
