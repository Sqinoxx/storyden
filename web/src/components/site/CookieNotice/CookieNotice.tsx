"use client";

import React, { useEffect, useState } from "react";
import { Cookie, X, Check, Info, ShieldCheck } from "lucide-react";
import { styled, Box, HStack, Stack } from "@/styled-system/jsx";
import { Portal } from "@ark-ui/react";

const COOKIE_CONSENT_KEY = "storyden_cookie_consent";

export function CookieNotice() {
  const [isVisible, setIsVisible] = useState(false);
  const [showDetails, setShowDetails] = useState(false);
  const [isAnimatingOut, setIsAnimatingOut] = useState(false);
  // Listen to custom open event (e.g. from footer/sidebar Cookie link)
  useEffect(() => {
    const handleReopen = () => {
      setIsAnimatingOut(false);
      setIsVisible(true);
    };

    window.addEventListener("open-cookie-settings", handleReopen);
    return () => window.removeEventListener("open-cookie-settings", handleReopen);
  }, []);

  const handleAccept = () => {
    setIsAnimatingOut(true);
    setTimeout(() => {
      localStorage.setItem(COOKIE_CONSENT_KEY, "accepted");
      setIsVisible(false);
      setIsAnimatingOut(false);
    }, 200);
  };

  const handleCloseOnly = () => {
    setIsAnimatingOut(true);
    setTimeout(() => {
      localStorage.setItem(COOKIE_CONSENT_KEY, "essential_only");
      setIsVisible(false);
      setIsAnimatingOut(false);
    }, 200);
  };

  if (!isVisible) return null;

  return (
    <Portal>
      <styled.div
        position="fixed"
        bottom="5"
        right="5"
        style={{
          zIndex: 99990,
          animation: isAnimatingOut
            ? "cookieFadeOut 0.2s ease-in forwards"
            : "cookieSlideUp 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards",
        }}
      >
        <style jsx global>{`
          @keyframes cookieSlideUp {
            from {
              opacity: 0;
              transform: translateY(20px) scale(0.95);
            }
            to {
              opacity: 1;
              transform: translateY(0) scale(1);
            }
          }
          @keyframes cookieFadeOut {
            from {
              opacity: 1;
              transform: translateY(0) scale(1);
            }
            to {
              opacity: 0;
              transform: translateY(12px) scale(0.95);
            }
          }
        `}</style>

        <styled.div
          borderRadius="2xl"
          bgColor="bg.default"
          borderColor="border.subtle"
          borderWidth="thin"
          boxShadow="2xl"
          p="5"
          display="flex"
          flexDirection="column"
          gap="3.5"
          style={{
            width: "min(380px, calc(100vw - 32px))",
            backdropFilter: "blur(12px)",
            WebkitBackdropFilter: "blur(12px)",
          }}
        >
          {/* Header */}
          <HStack justifyContent="space-between" alignItems="center">
            <HStack gap="2.5" alignItems="center">
              <Box p="2" borderRadius="xl" bg="bg.subtle" color="fg.default">
                <Cookie size={18} />
              </Box>
              <styled.span fontWeight="semibold" fontSize="sm" color="fg.default">
                Cookie-Hinweis
              </styled.span>
            </HStack>

            <styled.button
              type="button"
              onClick={handleCloseOnly}
              display="flex"
              alignItems="center"
              justifyContent="center"
              p="1"
              borderRadius="md"
              color="fg.subtle"
              cursor="pointer"
              bg="transparent"
              transition="colors"
              _hover={{
                bg: "bg.muted",
                color: "fg.default",
              }}
              title="Schließen"
            >
              <X size={16} />
            </styled.button>
          </HStack>

          {/* Description */}
          <styled.p fontSize="xs" color="fg.subtle" lineHeight="relaxed">
            Wir nutzen ausschließlich <strong>technisch notwendige Cookies</strong> (z. B. für deine Anmeldung und Einstellungen). Kein Tracking, keine Werbe-Cookies.
          </styled.p>

          {/* Collapsible Details */}
          {showDetails && (
            <Stack
              gap="2"
              pt="2"
              pb="1"
              borderTopWidth="thin"
              borderColor="border.subtle"
              fontSize="xs"
              color="fg.subtle"
            >
              <HStack gap="2" alignItems="center" color="fg.default" fontWeight="medium">
                <ShieldCheck size={14} />
                <span>Essenzielle Cookies:</span>
              </HStack>
              <Stack gap="1" pl="5">
                <styled.div>• <strong>session</strong>: Hält dich eingeloggt</styled.div>
                <styled.div>• <strong>storyden-theme</strong>: Speichert dein Farbschema (Dunkel/Hell)</styled.div>
                <styled.div>• <strong>storyden_cookie_consent</strong>: Merkt sich diesen Hinweis</styled.div>
              </Stack>
            </Stack>
          )}

          {/* Action Buttons */}
          <HStack gap="2" pt="1" justifyContent="space-between">
            <styled.button
              type="button"
              onClick={() => setShowDetails(!showDetails)}
              fontSize="xs"
              color="fg.subtle"
              bg="transparent"
              border="none"
              cursor="pointer"
              px="2"
              py="1.5"
              borderRadius="md"
              display="flex"
              alignItems="center"
              gap="1"
              transition="colors"
              _hover={{
                color: "fg.default",
                bg: "bg.subtle",
              }}
            >
              <Info size={13} />
              {showDetails ? "Weniger" : "Details"}
            </styled.button>

            <HStack gap="2">
              <styled.button
                type="button"
                onClick={handleAccept}
                fontSize="xs"
                fontWeight="medium"
                color="fg.default"
                bg="bg.subtle"
                borderWidth="thin"
                borderColor="border.subtle"
                px="3.5"
                py="1.5"
                borderRadius="lg"
                cursor="pointer"
                display="flex"
                alignItems="center"
                gap="1.5"
                transition="colors"
                _hover={{
                  bg: "bg.muted",
                }}
              >
                <Check size={14} />
                Verstanden
              </styled.button>
            </HStack>
          </HStack>
        </styled.div>
      </styled.div>
    </Portal>
  );
}

/** Helper function to re-trigger cookie settings dialog from anywhere */
export function openCookieSettings() {
  if (typeof window !== "undefined") {
    window.dispatchEvent(new CustomEvent("open-cookie-settings"));
  }
}
