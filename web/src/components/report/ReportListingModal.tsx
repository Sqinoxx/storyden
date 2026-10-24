"use client";

import { DatagraphItemKind, Listing } from "@/api/openapi-schema";
import { MemberBadge } from "@/components/member/MemberBadge/MemberBadge";
import { useTranslation } from "@/lib/i18n";
import { VStack, styled } from "@/styled-system/jsx";

import { ReportModal, ReportModalProps } from "./ReportModal";

type Props = Omit<
  ReportModalProps,
  | "title"
  | "description"
  | "subject"
  | "targetId"
  | "targetKind"
  | "submitLabel"
  | "successMessage"
  | "loadingMessage"
> & {
  listing: Listing;
};

export function ReportListingModal({ listing, ...disclosure }: Props) {
  const t = useTranslation();
  return (
    <ReportModal
      title={t.marketplace.detail.reportTitle}
      description={t.marketplace.detail.reportDescription}
      subject={
        <VStack alignItems="start" gap="2">
          <styled.span fontWeight="medium" maxW="64" wordBreak="break-word">
            {listing.title}
          </styled.span>
          <MemberBadge
            profile={listing.owner}
            size="sm"
            name="full-horizontal"
          />
        </VStack>
      }
      targetId={listing.id}
      targetKind={DatagraphItemKind.listing}
      successMessage={t.marketplace.detail.reportSuccess}
      {...disclosure}
    />
  );
}
