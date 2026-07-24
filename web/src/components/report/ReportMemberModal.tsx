"use client";

import { DatagraphItemKind, ProfileReference } from "@/api/openapi-schema";
import { MemberBadge } from "@/components/member/MemberBadge/MemberBadge";

import { ReportModal, ReportModalProps } from "./ReportModal";
import { useTranslation } from "@/lib/i18n";

export type ReportMemberModalProps = Omit<ReportModalProps, "title" | "description" | "subject" | "targetId" | "targetKind" | "submitLabel" | "successMessage" | "loadingMessage"> & {
  profile: ProfileReference;
};

export function ReportMemberModal({ profile, ...disclosure }: ReportMemberModalProps) {
  const t = useTranslation();
  return (
    <ReportModal
      title={t.report.reportMemberTitle.replace("{name}", profile.name)}
      description={t.report.reportMemberDesc}
      subject={<MemberBadge profile={profile} name="full-vertical" />}
      targetId={profile.id}
      targetKind={DatagraphItemKind.profile}
      submitLabel={t.report.reportMemberSubmit}
      successMessage={t.report.reportMemberSuccess}
      loadingMessage={t.report.reportMemberLoading}
      {...disclosure}
    />
  );
}
