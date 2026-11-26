# 销售日报 PDF 字体

导出「销售日报 PDF」时，浏览器才按当前店面风格下载这里的一种字体，嵌入 PDF（pdfmake 只写入用到的字形）。首屏和其他页面都不会加载这些文件。

| 文件 | 用于 | 大小 |
|---|---|---|
| `noto-serif-sc-regular.subset.ttf` | classic、exhibition（衬线，对应屏幕上的宋体） | 3,709,948 B |
| `noto-sans-sc-regular.subset.ttf` | shop（无衬线，对应屏幕上的 Inter / 苹方） | 2,688,676 B |
| `coverage.json` | 两种字体都收录的码位，导出时把缺字换成 □ | 57,105 B |

风格到字体的映射只在 `src/lib/reports/pdf/theme-fonts.ts` 一处。宋体 3.71 MB 超过最初 3.5 MB 的预算，已由 Kel 于 2026-11-26 确认接受。

## 收录范围

- 《通用规范汉字表》（2013）8105 字：一级 3500、二级 3000、三级 1605。
- 可打印 ASCII、Latin-1、Latin Extended-A（马来文和常见欧洲拉丁字母）。
- 常用标点、CJK 标点、全角字符、货币符号（RM 用 ASCII，S$、¥、€ 等），以及缺字替代符 □（U+25A1）。

日文假名、繁体专用字、谚文、emoji 和表外生僻字都不收录。导出时这些字符会显示为 □，并在 PDF 页脚注明。零宽连接符、变体选择符和肤色修饰符会被去掉，所以一个组合 emoji 只按它的基础字形各显示一个 □。

## 来源与许可证

- 字体：Google Fonts 官方仓库的 Noto Sans SC / Noto Serif SC 可变字体，固定 commit 和 SHA-256（见 `coverage.json` 与脚本）。SIL Open Font License 1.1，全文见 `OFL.txt`，版权声明见 `COPYRIGHT.txt`。OFL 允许把字体子集嵌入 PDF，生成的 PDF 不受 OFL 约束；随网站分发的子集属于 Modified Version，保留 OFL 和版权声明，不使用保留名 “Source”。
- 字表：Unicode Character Database **Unihan 17.0.0** 的 `kTGH` 字段（`https://www.unicode.org/Public/17.0.0/ucd/Unihan.zip`，SHA-256 `f7a48b2b545acfaa77b2d607ae28747404ce02baefee16396c5d2d7a8ef34b5e`），Unicode License V3，全文见 `UNICODE-LICENSE.txt`。`kTGH` 是 Unicode 联盟给《通用规范汉字表》每个字标注的序号。已核实：8105 个码位，序号 1–8105 不重不漏，三级字数为 3500 / 3000 / 1605；与之前草稿用过的 shengdoushi/common-standard-chinese-characters-table@d9b599a（无 LICENSE，已弃用）以及 npm `togscc@1.3.1` 逐字比对，三者完全一致。

## 重新生成

只有在字表、字体版本或收录范围变化时才需要重新生成；`npm run build` 不会运行脚本，也不联网。

```sh
python3 -m pip install fonttools   # 只需在生成字体的机器上安装
python3 web/scripts/subset-report-fonts.py
```

脚本会下载固定版本的上游文件并校验 SHA-256，把可变字体固定到 Regular（wght 400）、去掉 hinting，然后写出两个 `.subset.ttf`、`coverage.json`、`OFL.txt` 和 `UNICODE-LICENSE.txt`。重新生成后：

1. 把 `coverage.json` 里两个字体的 `bytes` 和 `sha256` 同步到 `src/lib/reports/pdf/theme-fonts.ts`，并把 `REPORT_COVERAGE_URL` 的版本号改成新 `coverage.json` 的 SHA-256 前 12 位。`tests/report-pdf.test.cjs` 会核对这几处是否一致。
2. 更新本文件里的大小。
