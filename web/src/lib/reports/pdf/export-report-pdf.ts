import type { ReportDocument } from "../document-model";
import { applyGlyphCoverage, createGlyphCoverage, type CoverageRanges } from "../glyph-coverage";
import { reportPdfFilename } from "../report-filename";
import { buildPdfDefinition, REPORT_PDF_FONT } from "./pdf-definition";
import { REPORT_COVERAGE_URL, reportColorsFor, reportFontFor, type ReportFontFace } from "./theme-fonts";

// Runs only after the export click. pdfmake (~1 MB) and the 2.7–3.7 MB font
// are loaded here, never on page load. Successful downloads are kept for the
// rest of the session; a failed one is dropped so the next click retries.

export type ExportStage =
  | { kind: "library" }
  | { kind: "font"; loaded: number; total: number }
  | { kind: "render" };

export class ReportExportError extends Error {}

const FETCH_TIMEOUT_MS = 60_000;
const REVOKE_AFTER_MS = 60_000;

type PdfMakeInstance = typeof import("pdfmake");

let libraryPromise: Promise<PdfMakeInstance> | null = null;
let coveragePromise: Promise<CoverageRanges> | null = null;
const fontPromises = new Map<string, Promise<Uint8Array>>();

function remember<T>(promise: Promise<T>, forget: () => void): Promise<T> {
  promise.catch(forget);
  return promise;
}

function loadLibrary(): Promise<PdfMakeInstance> {
  libraryPromise ??= remember(
    import("pdfmake").then((module) => {
      const instance = ((module as { default?: PdfMakeInstance }).default ?? module) as PdfMakeInstance;
      // The report never references images or URLs; refuse any remote fetch.
      instance.setUrlAccessPolicy(() => false);
      return instance;
    }),
    () => { libraryPromise = null; },
  );
  return libraryPromise;
}

async function fetchWithTimeout(url: string): Promise<Response> {
  const controller = new AbortController();
  const timer = window.setTimeout(() => controller.abort(), FETCH_TIMEOUT_MS);
  try {
    const response = await fetch(url, { signal: controller.signal, credentials: "same-origin" });
    if (!response.ok) throw new ReportExportError(`字体文件下载失败（HTTP ${response.status}），请稍后重试。`);
    return response;
  } finally {
    window.clearTimeout(timer);
  }
}

function loadCoverage(): Promise<CoverageRanges> {
  coveragePromise ??= remember(
    fetchWithTimeout(REPORT_COVERAGE_URL)
      .then((response) => response.json() as Promise<{ ranges?: unknown }>)
      .then(({ ranges }) => {
        if (!Array.isArray(ranges)) throw new ReportExportError("字体字表无法读取，请刷新页面后重试。");
        return ranges as CoverageRanges;
      }),
    () => { coveragePromise = null; },
  );
  return coveragePromise;
}

async function downloadFont(face: ReportFontFace, onProgress: (loaded: number) => void): Promise<Uint8Array> {
  const response = await fetchWithTimeout(face.url);
  // Content-Length may be the compressed size, so progress counts decoded
  // bytes against the known file size instead.
  const reader = response.body?.getReader();
  if (!reader) {
    const bytes = new Uint8Array(await response.arrayBuffer());
    onProgress(bytes.byteLength);
    return bytes;
  }
  const bytes = new Uint8Array(face.bytes);
  let loaded = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    if (loaded + value.byteLength > face.bytes) throw new ReportExportError("字体文件与预期不符，请刷新页面后重试。");
    bytes.set(value, loaded);
    loaded += value.byteLength;
    onProgress(loaded);
  }
  if (loaded !== face.bytes) throw new ReportExportError("字体文件下载不完整，请重试。");
  return bytes;
}

function loadFont(face: ReportFontFace, onProgress: (loaded: number) => void): Promise<Uint8Array> {
  let promise = fontPromises.get(face.url);
  if (!promise) {
    promise = remember(downloadFont(face, onProgress), () => fontPromises.delete(face.url));
    fontPromises.set(face.url, promise);
  } else {
    onProgress(face.bytes);
  }
  return promise;
}

function saveBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.rel = "noopener";
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  // Safari may still be reading the blob right after click().
  window.setTimeout(() => URL.revokeObjectURL(url), REVOKE_AFTER_MS);
}

export function describeExportError(error: unknown): string {
  if (error instanceof ReportExportError) return error.message;
  if (error instanceof DOMException && error.name === "AbortError") return "字体下载超时，请检查网络后重试。";
  if (error instanceof TypeError) return "无法下载 PDF 组件，请检查网络后重试。";
  return "PDF 生成失败，请重试。";
}

export type ExportReportPdfResult = { filename: string; replacedGlyphs: number };

export async function exportReportPdf(
  doc: ReportDocument,
  theme: string,
  onStage: (stage: ExportStage) => void,
): Promise<ExportReportPdfResult> {
  const face = reportFontFor(theme);
  onStage({ kind: "library" });
  const libraryLoading = loadLibrary();
  const coverageLoading = loadCoverage();
  const [pdfMake, ranges] = await Promise.all([libraryLoading, coverageLoading]);
  onStage({ kind: "font", loaded: 0, total: face.bytes });
  const font = await loadFont(face, (loaded) => onStage({ kind: "font", loaded, total: face.bytes }));

  onStage({ kind: "render" });
  const { value: covered, replaced } = applyGlyphCoverage(doc, createGlyphCoverage(ranges));
  const definition = buildPdfDefinition(covered, { colors: reportColorsFor(theme), missingGlyphs: replaced > 0 });
  // Only Regular ships; bold/italic requests reuse it instead of failing.
  pdfMake.setFonts({ [REPORT_PDF_FONT]: { normal: font, bold: font, italics: font, bolditalics: font } });
  const blob = await pdfMake.createPdf(definition).getBlob();
  const filename = reportPdfFilename(doc.dates, doc.currencyLabel);
  saveBlob(blob, filename);
  return { filename, replacedGlyphs: replaced };
}
