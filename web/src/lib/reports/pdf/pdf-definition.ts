import type { Content, TDocumentDefinitions } from "pdfmake/interfaces";
import type { ReportBlock, ReportDaySection, ReportDocument, ReportMetric, ReportRow, ReportSummary } from "../document-model";
import { MISSING_GLYPH } from "../glyph-coverage";
import type { ReportColors } from "./theme-fonts";

// Turns the report document model into a pdfmake definition. Only one font is
// registered (the theme's Regular subset), and the content carries no images,
// links or attachments, so the PDF holds nothing beyond the report text.

export const REPORT_PDF_FONT = "ReportFont";

export type PdfDefinitionOptions = { colors: ReportColors; missingGlyphs: boolean };

const thinLines = (colors: ReportColors) => ({
  hLineWidth: (index: number) => (index === 0 ? 0 : 0.5),
  vLineWidth: () => 0,
  hLineColor: () => colors.line,
  paddingLeft: (index: number) => (index === 0 ? 0 : 12),
  paddingRight: () => 0,
  paddingTop: () => 3.5,
  paddingBottom: () => 3.5,
});

function metricCell(metric: ReportMetric, colors: ReportColors): Content {
  return {
    stack: [
      { text: metric.label, fontSize: 8, color: colors.muted },
      { text: metric.value, fontSize: metric.lead ? 18 : 14, color: metric.lead ? colors.accent : colors.ink, margin: [0, 3, 0, 2] },
      { text: metric.note, fontSize: 7.5, color: colors.muted },
    ],
  };
}

function metricsGrid(metrics: ReportMetric[], colors: ReportColors): Content {
  const rows: Content[][] = [];
  for (let index = 0; index < metrics.length; index += 3) {
    const row = metrics.slice(index, index + 3).map((metric) => metricCell(metric, colors));
    while (row.length < 3) row.push({ text: "" });
    rows.push(row);
  }
  return {
    table: { widths: ["*", "*", "*"], body: rows },
    layout: {
      hLineWidth: () => 0.5,
      vLineWidth: (index: number, node) => (index === 0 || index === (node.table.widths?.length ?? 3) ? 0 : 0.5),
      hLineColor: () => colors.line,
      vLineColor: () => colors.line,
      paddingLeft: () => 8,
      paddingRight: () => 8,
      paddingTop: () => 6,
      paddingBottom: () => 6,
    },
    margin: [0, 10, 0, 2],
  };
}

function rowsTable(rows: ReportRow[], colors: ReportColors): Content {
  const columns = rows[0]?.cells.length ?? 1;
  const widths = ["*", ...Array.from({ length: columns - 1 }, () => "auto"), "auto"];
  return {
    table: {
      widths,
      dontBreakRows: true,
      body: rows.map((row) => [
        ...row.cells.map((cell, index): Content => ({ text: cell, color: index === 0 ? colors.ink : colors.muted, fontSize: 9 })),
        { text: row.amount, alignment: "right", color: colors.ink, fontSize: 9.5 },
      ]),
    },
    layout: thinLines(colors),
  };
}

function blockHead(title: string, caption: string, colors: ReportColors): Content {
  return {
    columns: [
      { text: title, fontSize: 11.5, color: colors.ink, width: "*" },
      { text: caption, fontSize: 7.5, color: colors.muted, width: "auto", alignment: "right", margin: [0, 3, 0, 0] },
    ],
    margin: [0, 12, 0, 2],
  };
}

function summarySection(summary: ReportSummary, colors: ReportColors): Content {
  return { stack: [blockHead(summary.title, summary.caption, colors), rowsTable(summary.rows, colors)] };
}

function detailBlock(block: ReportBlock, colors: ReportColors): Content {
  const body: Content[] = [blockHead(block.title, block.caption, colors)];
  if (block.groups.length === 0) {
    body.push({ text: block.empty, fontSize: 9, color: colors.muted, margin: [0, 2, 0, 0] });
  }
  block.groups.forEach((group) => {
    // Keep a short currency group with its label; long ones may still split.
    body.push({
      unbreakable: group.rows.length <= 12,
      stack: [
        { text: group.currency, fontSize: 8, color: colors.accent, margin: [0, 5, 0, 0] },
        rowsTable(group.rows, colors),
      ],
    });
  });
  return { stack: body };
}

function daySection(day: ReportDaySection, colors: ReportColors, first: boolean): Content {
  const breakBefore = first ? {} : { pageBreak: "before" as const };
  if (day.missing) {
    return {
      ...breakBefore,
      stack: [
        { text: day.date, fontSize: 9, color: colors.muted },
        { text: day.message, fontSize: 12, color: colors.ink, margin: [0, 8, 0, 0] },
      ],
    };
  }
  const content: Content[] = [
    {
      columns: [
        {
          width: "*",
          stack: [
            { text: day.eyebrow, fontSize: 8.5, color: colors.muted },
            { text: day.heading, fontSize: 19, color: colors.ink, margin: [0, 3, 0, 3] },
            { text: day.generated, fontSize: 8, color: colors.muted },
          ],
        },
        { width: "auto", text: day.stamp, fontSize: 8.5, color: colors.accent, margin: [0, 2, 0, 0] },
      ],
    },
    metricsGrid(day.metrics, colors),
  ];
  if (day.currencySummary) content.push(summarySection(day.currencySummary, colors));
  day.blocks.forEach((block) => content.push(detailBlock(block, colors)));
  return { ...breakBefore, stack: content };
}

export function buildPdfDefinition(doc: ReportDocument, { colors, missingGlyphs }: PdfDefinitionOptions): TDocumentDefinitions {
  return {
    pageSize: "A4",
    pageMargins: [44, 52, 44, 48],
    info: { title: doc.title, creator: "Stallink", producer: "Stallink" },
    defaultStyle: { font: REPORT_PDF_FONT, fontSize: 10, color: colors.ink, lineHeight: 1.25 },
    header: () => ({
      columns: [
        { text: doc.title, fontSize: 8, color: colors.muted },
        { text: `币种：${doc.currencyLabel}`, fontSize: 8, color: colors.muted, alignment: "right" },
      ],
      margin: [44, 26, 44, 0],
    }),
    footer: (currentPage: number, pageCount: number) => ({
      columns: [
        missingGlyphs
          ? { text: `字体未收录的字符已显示为 ${MISSING_GLYPH}，原文请以系统内数据为准。`, fontSize: 7.5, color: colors.muted }
          : { text: "" },
        { text: `第 ${currentPage} / ${pageCount} 页`, fontSize: 7.5, color: colors.muted, alignment: "right", width: "auto" },
      ],
      margin: [44, 18, 44, 0],
    }),
    content: doc.days.map((day, index) => daySection(day, colors, index === 0)),
  };
}
