"use client";

import React, { useEffect } from "react";
import { Portal } from "@ark-ui/react";
import { X, ShieldCheck, Mail, MapPin, User, FileText, Scale } from "lucide-react";
import { styled, Box, HStack, Stack } from "@/styled-system/jsx";

interface ImpressumModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function ImpressumModal({ isOpen, onClose }: ImpressumModalProps) {
  // Close on Escape key press
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <Portal>
      {/* Backdrop */}
      <styled.div
        position="fixed"
        onClick={onClose}
        style={{
          top: "-20px",
          left: "-20px",
          right: "-20px",
          bottom: "-20px",
          zIndex: 99998,
          backgroundColor: "rgba(0, 0, 0, 0.6)",
          backdropFilter: "blur(6px)",
          WebkitBackdropFilter: "blur(6px)",
          transform: "translateZ(0)",
          cursor: "default",
          animation: "impressumFadeIn 0.15s ease-out forwards",
        }}
      />

      {/* Modal Container */}
      <styled.div
        position="fixed"
        display="flex"
        flexDirection="column"
        borderRadius="2xl"
        bgColor="bg.default"
        borderColor="border.subtle"
        borderWidth="thin"
        boxShadow="2xl"
        overflow="hidden"
        style={{
          zIndex: 99999,
          top: "50%",
          left: "50%",
          transform: "translate(-50%, -50%) translateZ(0)",
          width: "min(680px, 92vw)",
          maxHeight: "85vh",
          animation: "impressumZoomIn 0.15s cubic-bezier(0.16, 1, 0.3, 1) forwards",
        }}
      >
        <style jsx global>{`
          @keyframes impressumFadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
          }
          @keyframes impressumZoomIn {
            from { opacity: 0; transform: translate(-50%, -48%) scale(0.97) translateZ(0); }
            to { opacity: 1; transform: translate(-50%, -50%) scale(1) translateZ(0); }
          }
        `}</style>

        {/* Modal Header */}
        <styled.div
          display="flex"
          alignItems="center"
          justifyContent="space-between"
          px="6"
          py="4"
          borderBottomWidth="thin"
          borderColor="border.subtle"
          bg="bg.subtle"
          flexShrink="0"
        >
          <HStack gap="2.5" alignItems="center">
            <Scale size={20} />
            <styled.h2 fontSize="lg" fontWeight="semibold" color="fg.default">
              Impressum
            </styled.h2>
          </HStack>

          <styled.button
            type="button"
            onClick={onClose}
            display="flex"
            alignItems="center"
            justifyContent="center"
            p="1.5"
            borderRadius="md"
            color="fg.subtle"
            cursor="pointer"
            bg="transparent"
            transition="colors"
            _hover={{
              bg: "bg.muted",
              color: "fg.default",
            }}
            aria-label="Schließen"
          >
            <X size={18} />
          </styled.button>
        </styled.div>

        {/* Modal Content / Body */}
        <styled.div
          p="6"
          overflowY="auto"
          display="flex"
          flexDirection="column"
          gap="6"
          fontSize="sm"
          lineHeight="relaxed"
          color="fg.default"
        >
          {/* Angaben gemäß § 5 DDG */}
          <Stack gap="3">
            <HStack gap="2" color="fg.default" fontWeight="medium">
              <ShieldCheck size={16} />
              <styled.h3 fontSize="md" fontWeight="semibold" color="fg.default">
                Angaben gemäß § 5 DDG
              </styled.h3>
            </HStack>
            <Box
              p="4"
              borderRadius="lg"
              bg="bg.subtle"
              borderWidth="thin"
              borderColor="border.subtle"
              display="flex"
              flexDirection="column"
              gap="2.5"
            >
              <HStack gap="2.5" alignItems="flex-start">
                <User size={15} style={{ marginTop: "3px", flexShrink: 0 }} />
                <Box>
                  <styled.span fontWeight="medium">Dienstanbieter / Betreiber:</styled.span>{" "}
                  Zahnis Regensburg
                </Box>
              </HStack>

              <HStack gap="2.5" alignItems="flex-start">
                <MapPin size={15} style={{ marginTop: "3px", flexShrink: 0 }} />
                <Box>
                  <styled.span fontWeight="medium">Anschrift:</styled.span>{" "}
                  Franz Josef Strauß Allee 11, 93053 Regensburg
                </Box>
              </HStack>

              <HStack gap="2.5" alignItems="flex-start">
                <Mail size={15} style={{ marginTop: "3px", flexShrink: 0 }} />
                <Box>
                  <styled.span fontWeight="medium">Kontakt:</styled.span>{" "}
                  info@zahnmedizin-rgbg.de
                </Box>
              </HStack>
            </Box>
          </Stack>

          {/* Vertretungsberechtigte & Inhaltliche Verantwortung */}
          <Stack gap="3">
            <HStack gap="2" color="fg.default" fontWeight="medium">
              <FileText size={16} />
              <styled.h3 fontSize="md" fontWeight="semibold" color="fg.default">
                Verantwortlich für den Inhalt
              </styled.h3>
            </HStack>
            <Box
              p="4"
              borderRadius="lg"
              bg="bg.subtle"
              borderWidth="thin"
              borderColor="border.subtle"
            >
              <styled.p color="fg.subtle">
                Verantwortlich für den Inhalt nach § 18 Abs. 2 MStV:
              </styled.p>
              <styled.p fontWeight="medium" mt="1">
                Administration Zahnis Regensburg
              </styled.p>
              <styled.p fontSize="xs" color="fg.subtle" mt="1">
                Franz Josef Strauß Allee 11, 93053 Regensburg
              </styled.p>
            </Box>
          </Stack>

          {/* Haftungsausschluss (Disclaimer) */}
          <Stack gap="3">
            <styled.h3 fontSize="md" fontWeight="semibold" color="fg.default">
              Haftungsausschluss (Disclaimer)
            </styled.h3>

            <Box display="flex" flexDirection="column" gap="4">
              <Box>
                <styled.h4 fontWeight="semibold" mb="1" color="fg.default">
                  Haftung für Inhalte
                </styled.h4>
                <styled.p color="fg.subtle">
                  Als Diensteanbieter sind wir gemäß § 7 Abs. 1 DDG für eigene Inhalte auf diesen Seiten nach den allgemeinen Gesetzen verantwortlich. Nach §§ 8 bis 10 DDG sind wir als Diensteanbieter jedoch nicht verpflichtet, übermittelte oder gespeicherte fremde Informationen zu überwachen oder nach Umständen zu forschen, die auf eine rechtswidrige Tätigkeit hinweisen.
                </styled.p>
              </Box>

              <Box>
                <styled.h4 fontWeight="semibold" mb="1" color="fg.default">
                  Haftung für Links
                </styled.h4>
                <styled.p color="fg.subtle">
                  Unser Angebot enthält Links zu externen Websites Dritter, auf deren Inhalte wir keinen Einfluss haben. Deshalb können wir für diese fremden Inhalte auch keine Gewähr übernehmen. Für die Inhalte der verlinkten Seiten ist stets der jeweilige Anbieter oder Betreiber der Seiten verantwortlich.
                </styled.p>
              </Box>

              <Box>
                <styled.h4 fontWeight="semibold" mb="1" color="fg.default">
                  Urheberrecht
                </styled.h4>
                <styled.p color="fg.subtle">
                  Die durch die Seitenbetreiber erstellten Inhalte und Werke auf diesen Seiten unterliegen dem deutschen Urheberrecht. Die Vervielfältigung, Bearbeitung, Verbreitung und jede Art der Verwertung außerhalb der Grenzen des Urheberrechtes bedürfen der schriftlichen Zustimmung des jeweiligen Autors bzw. Erstellers.
                </styled.p>
              </Box>
            </Box>
          </Stack>

          {/* EU-Streitschlichtung */}
          <Box
            p="3.5"
            borderRadius="md"
            bg="bg.muted"
            fontSize="xs"
            color="fg.subtle"
          >
            <styled.span fontWeight="medium">Verbraucherstreitbeilegung:</styled.span> Wir sind nicht bereit oder verpflichtet, an Streitbeilegungsverfahren vor einer Verbraucherschlichtungsstelle teilzunehmen.
          </Box>
        </styled.div>
      </styled.div>
    </Portal>
  );
}
