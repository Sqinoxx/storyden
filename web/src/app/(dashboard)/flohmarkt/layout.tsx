import { notFound } from "next/navigation";
import { PropsWithChildren } from "react";

import { getServerSession } from "@/auth/server-session";
import { canSeeMarketplace } from "@/lib/marketplace/visibility";
import { getSettings } from "@/lib/settings/settings-server";

export default async function Layout({ children }: PropsWithChildren) {
  const [session, settings] = await Promise.all([
    getServerSession(),
    getSettings(),
  ]);

  if (!canSeeMarketplace(settings.marketplace_enabled, session)) {
    notFound();
  }

  return <>{children}</>;
}
