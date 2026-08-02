"use client";

import React from "react";
import { styled, Box, HStack, Stack } from "@/styled-system/jsx";
import { Scale, ShieldCheck, Mail, MapPin, User, FileText } from "lucide-react";
import { Breadcrumbs } from "@/components/ui/Breadcrumbs";

export function ImpressumPageScreen() {
  return (
    <Box p="6" maxW="4xl" mx="auto" w="full">
      <Box mb="4">
        <Breadcrumbs
          index={{ label: "Startseite", href: "/" }}
          crumbs={[{ label: "Impressum", href: "/impressum" }]}
        />
      </Box>

      <styled.div
        p="6"
        borderRadius="2xl"
        bgColor="bg.default"
        borderColor="border.subtle"
        borderWidth="thin"
        boxShadow="sm"
        display="flex"
        flexDirection="column"
        gap="6"
      >
        <HStack gap="3" alignItems="center" pb="4" borderBottomWidth="thin" borderColor="border.subtle">
          <Scale size={26} />
          <Box>
            <styled.h1 fontSize="2xl" fontWeight="bold" color="fg.default">
              Impressum
            </styled.h1>
            <styled.p fontSize="sm" color="fg.subtle">
              Rechtliche Informationen und Anbieterkennzeichnung gemäß § 5 DDG
            </styled.p>
          </Box>
        </HStack>

        {/* Angaben gemäß § 5 DDG */}
        <Stack gap="3">
          <HStack gap="2" color="fg.default">
            <ShieldCheck size={18} />
            <styled.h2 fontSize="lg" fontWeight="semibold" color="fg.default">
              Angaben gemäß § 5 DDG
            </styled.h2>
          </HStack>
          <Box
            p="5"
            borderRadius="xl"
            bg="bg.subtle"
            borderWidth="thin"
            borderColor="border.subtle"
            display="flex"
            flexDirection="column"
            gap="3"
            fontSize="sm"
          >
            <HStack gap="3" alignItems="flex-start">
              <User size={16} style={{ marginTop: "3px", flexShrink: 0 }} />
              <Box>
                <styled.span fontWeight="semibold">Dienstanbieter / Betreiber:</styled.span>{" "}
                Zahnis Regensburg
              </Box>
            </HStack>

            <HStack gap="3" alignItems="flex-start">
              <MapPin size={16} style={{ marginTop: "3px", flexShrink: 0 }} />
              <Box>
                <styled.span fontWeight="semibold">Anschrift:</styled.span>{" "}
                Franz Josef Strauß Allee 11, 93053 Regensburg
              </Box>
            </HStack>

            <HStack gap="3" alignItems="flex-start">
              <Mail size={16} style={{ marginTop: "3px", flexShrink: 0 }} />
              <Box>
                <styled.span fontWeight="semibold">Kontakt:</styled.span>{" "}
                info@zahnmedizin-rgbg.de
              </Box>
            </HStack>
          </Box>
        </Stack>

        {/* Vertretung & Inhaltliche Verantwortung */}
        <Stack gap="3">
          <HStack gap="2" color="fg.default">
            <FileText size={18} />
            <styled.h2 fontSize="lg" fontWeight="semibold" color="fg.default">
              Verantwortlich für den Inhalt
            </styled.h2>
          </HStack>
          <Box
            p="5"
            borderRadius="xl"
            bg="bg.subtle"
            borderWidth="thin"
            borderColor="border.subtle"
            fontSize="sm"
          >
            <styled.p color="fg.subtle">
              Verantwortlich für den Inhalt nach § 18 Abs. 2 MStV:
            </styled.p>
            <styled.p fontWeight="semibold" mt="1.5">
              Administration Zahnis Regensburg
            </styled.p>
            <styled.p fontSize="xs" color="fg.subtle" mt="1">
              Franz Josef Strauß Allee 11, 93053 Regensburg
            </styled.p>
          </Box>
        </Stack>

        {/* Haftungsausschluss (Disclaimer) */}
        <Stack gap="4">
          <styled.h2 fontSize="lg" fontWeight="semibold" color="fg.default">
            Haftungsausschluss (Disclaimer)
          </styled.h2>

          <Stack gap="4" fontSize="sm" color="fg.default">
            <Box>
              <styled.h3 fontWeight="semibold" mb="1">
                Haftung für Inhalte
              </styled.h3>
              <styled.p color="fg.subtle" lineHeight="relaxed">
                Als Diensteanbieter sind wir gemäß § 7 Abs. 1 DDG für eigene Inhalte auf diesen Seiten nach den allgemeinen Gesetzen verantwortlich. Nach §§ 8 bis 10 DDG sind wir als Diensteanbieter jedoch nicht verpflichtet, übermittelte oder gespeicherte fremde Informationen zu überwachen oder nach Umständen zu forschen, die auf eine rechtswidrige Tätigkeit hinweisen.
              </styled.p>
            </Box>

            <Box>
              <styled.h3 fontWeight="semibold" mb="1">
                Haftung für Links
              </styled.h3>
              <styled.p color="fg.subtle" lineHeight="relaxed">
                Unser Angebot enthält Links zu externen Websites Dritter, auf deren Inhalte wir keinen Einfluss haben. Deshalb können wir für diese fremden Inhalte auch keine Gewähr übernehmen. Für die Inhalte der verlinkten Seiten ist stets der jeweilige Anbieter oder Betreiber der Seiten verantwortlich.
              </styled.p>
            </Box>

            <Box>
              <styled.h3 fontWeight="semibold" mb="1">
                Urheberrecht
              </styled.h3>
              <styled.p color="fg.subtle" lineHeight="relaxed">
                Die durch die Seitenbetreiber erstellten Inhalte und Werke auf diesen Seiten unterliegen dem deutschen Urheberrecht. Die Vervielfältigung, Bearbeitung, Verbreitung und jede Art der Verwertung außerhalb der Grenzen des Urheberrechtes bedürfen der schriftlichen Zustimmung des jeweiligen Autors bzw. Erstellers.
              </styled.p>
            </Box>
          </Stack>
        </Stack>

        {/* EU-Streitschlichtung */}
        <Box
          p="4"
          borderRadius="lg"
          bg="bg.muted"
          fontSize="xs"
          color="fg.subtle"
        >
          <styled.span fontWeight="semibold">Verbraucherstreitbeilegung:</styled.span> Wir sind nicht bereit oder verpflichtet, an Streitbeilegungsverfahren vor einer Verbraucherschlichtungsstelle teilzunehmen.
        </Box>
      </styled.div>
    </Box>
  );
}
