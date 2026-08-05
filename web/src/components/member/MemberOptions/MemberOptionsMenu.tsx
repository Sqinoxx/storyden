import { MenuSelectionDetails, Portal } from "@ark-ui/react";
import Link from "next/link";
import { PropsWithChildren } from "react";
import { toast } from "sonner";

import { useSession } from "@/auth";

import { Permission, ProfileReference } from "@/api/openapi-schema";
import { ReportMemberMenuItem } from "@/components/report/ReportMemberMenuItem";
import * as Menu from "@/components/ui/menu";
import { WEB_ADDRESS } from "@/config";
import { hasPermission } from "@/utils/permissions";
import { useCopyToClipboard } from "@/utils/useCopyToClipboard";
import { useTranslation } from "@/lib/i18n";

import { MemberIdent } from "../MemberBadge/MemberIdent";
import { MemberPasswordResetTrigger } from "../MemberPasswordReset/MemberPasswordResetTrigger";
import { MemberDeletionTrigger } from "../MemberDeletion/MemberDeletionTrigger";
import { MemberRoleMenu } from "../MemberRoleMenu/MemberRoleMenu";
import { MemberSuspensionTrigger } from "../MemberSuspension/MemberSuspensionTrigger";
import { MemberWarningTrigger } from "../MemberWarning/MemberWarningTrigger";

export type Props = {
  profile: ProfileReference;
  asChild?: boolean;
};

export function MemberOptionsMenu({
  children,
  profile,
  ...props
}: PropsWithChildren<Props>) {
  const session = useSession();
  const [_, copy] = useCopyToClipboard();
  const t = useTranslation();

  const permalink = `${WEB_ADDRESS}/m/${profile.handle}`;

  const isSelf = session?.id === profile.id;

  const canWarn = !isSelf && hasPermission(session, Permission.MANAGE_WARNINGS);
  const canSuspend =
    !isSelf && hasPermission(session, Permission.MANAGE_SUSPENSIONS);
  const canResetPassword =
    !isSelf && hasPermission(session, Permission.MANAGE_ACCOUNTS);

  const isAdmin = session?.admin === true || hasPermission(session, Permission.ADMINISTRATOR);
  const canDelete = !isSelf && isAdmin && profile.suspended;

  const isRoleChangeEnabled = hasPermission(session, "MANAGE_ROLES");

  function handleSelect(value: MenuSelectionDetails) {
    switch (value.value) {
      case "copy-link":
        copy(permalink);
        toast("Link copied to clipboard");
        break;
    }
  }

  return (
    <Menu.Root onSelect={handleSelect}>
      <Menu.Trigger
        className="member-options-menu__trigger"
        maxW="full"
        cursor="pointer"
        asChild={props.asChild}
        textAlign="start"
      >
        {children}
      </Menu.Trigger>

      <Portal>
        <Menu.Positioner>
          <Menu.Content minW="48" userSelect="none">
            <Menu.ItemGroup id="group">
              <Menu.ItemGroupLabel display="flex" gap="2" alignItems="center">
                <MemberIdent profile={profile} size="md" name="full-vertical" />
              </Menu.ItemGroupLabel>

              <Menu.Separator />

              <Menu.ItemGroup>
                <Link href={permalink}>
                  <Menu.Item value="view">{t.actions.viewProfile}</Menu.Item>
                </Link>
              </Menu.ItemGroup>

              <Menu.Item value="copy-link">{t.actions.copyLink}</Menu.Item>

              <ReportMemberMenuItem profile={profile} />

              {isRoleChangeEnabled && <MemberRoleMenu profile={profile} />}

              {canResetPassword && (
                <MemberPasswordResetTrigger profile={profile}>
                  <Menu.Item value="reset-password">{t.actions.resetPassword}</Menu.Item>
                </MemberPasswordResetTrigger>
              )}

              {canWarn && (
                <MemberWarningTrigger profile={profile}>
                  <Menu.Item
                    value="warn"
                    color="fg.warning"
                    _hover={{ color: "fg.warning", background: "bg.warning" }}
                  >
                    {t.actions.warnMember}
                  </Menu.Item>
                </MemberWarningTrigger>
              )}

              {canSuspend && (
                <MemberSuspensionTrigger profile={profile}>
                  <Menu.Item
                    value="suspend"
                    color="fg.destructive"
                    _hover={{
                      color: "fg.destructive",
                      background: "bg.destructive",
                    }}
                  >
                    {profile.suspended ? t.actions.unsuspend : t.actions.suspend}
                  </Menu.Item>
                </MemberSuspensionTrigger>
              )}

              {canDelete && (
                <MemberDeletionTrigger profile={profile}>
                  <Menu.Item
                    value="delete-member"
                    color="fg.destructive"
                    _hover={{
                      color: "fg.destructive",
                      background: "bg.destructive",
                    }}
                  >
                    {t.actions.deleteMember}
                  </Menu.Item>
                </MemberDeletionTrigger>
              )}
            </Menu.ItemGroup>
          </Menu.Content>
        </Menu.Positioner>
      </Portal>
    </Menu.Root>
  );
}
