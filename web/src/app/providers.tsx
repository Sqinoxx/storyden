"use client";

import { PropsWithChildren } from "react";
import { Toaster } from "sonner";
import { SWRConfig } from "swr";

import { AuthProvider } from "@/auth/AuthProvider";

import { CookieNotice } from "@/components/site/CookieNotice/CookieNotice";
import { useCacheProvider } from "@/lib/cache/swr-cache";
import { DndProvider } from "@/lib/dragdrop/provider";
import { LanguageProvider } from "@/lib/i18n";
import { ThemeProvider } from "@/lib/theme/ThemeProvider";

export function Providers({ children }: PropsWithChildren) {
  const provider = useCacheProvider();

  return (
    <ThemeProvider>
      <LanguageProvider>
        <AuthProvider>
          <SWRConfig
            value={{
              keepPreviousData: true,
              // provider: provider,
            }}
          >
            <DndProvider>
              <Toaster />
              <CookieNotice />

              {/* -- */}
              {children}
              {/* -- */}
            </DndProvider>
          </SWRConfig>
        </AuthProvider>
      </LanguageProvider>
    </ThemeProvider>
  );
}
