import { getAPIAddress } from "@/config";

export const revalidate = 3600;

export async function GET() {
  const res = await fetch(`${getAPIAddress()}/api/info/icon/512x512`, {
    next: { revalidate },
  });

  if (!res.ok) {
    return new Response(null, { status: res.status });
  }

  return new Response(await res.arrayBuffer(), {
    headers: {
      "Content-Type": res.headers.get("Content-Type") ?? "image/png",
      "Cache-Control": "public, max-age=3600, stale-while-revalidate=86400",
    },
  });
}
