import { NextRequest, NextResponse } from "next/server";
import { apiOrigin } from "@/lib/api";

async function proxy(req: NextRequest, path: string[]) {
  const url = new URL(`/api/${path.map(encodeURIComponent).join("/")}${req.nextUrl.search}`, apiOrigin());
  const headers = new Headers(req.headers);
  headers.delete("host");
  const init: RequestInit & { duplex?: "half" } = {
    method: req.method,
    headers,
    redirect: "manual",
  };
  if (req.method !== "GET" && req.method !== "HEAD") {
    init.body = req.body;
    init.duplex = "half";
  }
  const res = await fetch(url, init);
  const out = new NextResponse(res.body, { status: res.status });
  res.headers.forEach((value, key) => {
    const name = key.toLowerCase();
    if (name === "set-cookie" || name === "content-encoding" || name === "content-length") {
      return;
    }
    out.headers.set(key, value);
  });
  for (const cookie of res.headers.getSetCookie()) {
    out.headers.append("set-cookie", cookie);
  }
  return out;
}

type Ctx = { params: Promise<{ path: string[] }> };

export function GET(req: NextRequest, ctx: Ctx) {
  return ctx.params.then((params) => proxy(req, params.path));
}

export function POST(req: NextRequest, ctx: Ctx) {
  return ctx.params.then((params) => proxy(req, params.path));
}

export function PATCH(req: NextRequest, ctx: Ctx) {
  return ctx.params.then((params) => proxy(req, params.path));
}

export function PUT(req: NextRequest, ctx: Ctx) {
  return ctx.params.then((params) => proxy(req, params.path));
}

export function DELETE(req: NextRequest, ctx: Ctx) {
  return ctx.params.then((params) => proxy(req, params.path));
}
