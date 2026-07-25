"use client";

import { PropsWithChildren } from "react";
import { Toaster } from "sonner";
import { SWRConfig } from "swr";

import { AuthProvider } from "@/auth/AuthProvider";

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
