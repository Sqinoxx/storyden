"use client";

import { formatDistanceToNow } from "date-fns";
import { de, enUS } from "date-fns/locale";

import { useLanguage } from "@/lib/i18n";
import { Unready } from "@/components/site/Unready";

import { ContentFormField } from "@/components/content/ContentComposer/ContentField";
import { MemberAvatar } from "@/components/member/MemberBadge/MemberAvatar";
import { MemberIdent } from "@/components/member/MemberBadge/MemberIdent";
import { MemberOptionsMenu } from "@/components/member/MemberOptions/MemberOptionsMenu";
import { ProfileAccountManagement } from "@/components/profile/ProfileAccountManagement/ProfileAccountManagement";
import { ProfileContent } from "@/components/profile/ProfileContent/ProfileContent";
import { ProfileSuspendedBanner } from "@/components/profile/ProfileSuspendedBanner";
import { RoleBadgeList } from "@/components/role/RoleBadge/RoleBadgeList";
import { EditAction } from "@/components/site/Action/Edit";
import { MoreAction } from "@/components/site/Action/More";
import { SaveAction } from "@/components/site/Action/Save";
import { DotSeparator } from "@/components/site/Dot";
import { LikeIcon } from "@/components/ui/icons/Like";
import { Input } from "@/components/ui/input";
import {
  SEMESTER_OPTIONS,
  formatSemester,
  parseAcademicMeta,
} from "@/lib/profile/academic";
import {
  Box,
  CardBox,
  Flex,
  HStack,
  LStack,
  styled,
} from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { Form, Props, useProfileScreen } from "./useProfileScreen";

export function ProfileScreen(props: Props) {
  const { ready, error, form, state, data, handlers } = useProfileScreen(props);
  const { language, t } = useLanguage();
  const dateLocale = language === "de" ? de : enUS;

  if (!ready) {
    return <Unready error={error} />;
  }

  const { session, profile } = data;
  const { isSelf, isEditing, canViewAccount, signaturesEnabled } = state;
  const semester = parseAcademicMeta(profile.meta)?.semester;
  const isEmpty =
    !profile.bio || profile.bio === "" || profile.bio === "<body></body>";
  const isSignatureEmpty =
    !profile.signature ||
    profile.signature === "" ||
    profile.signature === "<body></body>";

  return (
    <LStack w="full">
      <CardBox p="0">
        <styled.form className={lstack()} p="3" onSubmit={handlers.handleSave}>
          <Flex
            direction={{ base: "column-reverse", sm: "row" }}
            w="full"
            justify="space-between"
            alignItems={{ base: "end", sm: "start" }}
          >
            {isEditing ? (
              <HStack w="full" pr={{ base: "0", sm: "24" }}>
                <MemberAvatar
                  profile={profile}
                  size="lg"
                  editable={isEditing}
                />
                <LStack gap="1">
                  <LStack gap="0">
                    <Input
                      maxW={{ base: "full", sm: "64" }}
                      size="sm"
                      height="7"
                      px="2"
                      borderBottomRadius="none"
                      fontWeight="bold"
                      {...form.register("name")}
                    />
                    <Input
                      maxW={{ base: "full", sm: "64" }}
                      size="sm"
                      height="7"
                      px="2"
                      borderTop="none"
                      borderTopRadius="none"
                      {...form.register("handle")}
                    />
                  </LStack>
                  <RoleBadgeList roles={profile.roles} />
                </LStack>
              </HStack>
            ) : (
              <MemberIdent
                profile={profile}
                size="lg"
                name="full-vertical"
                showRoles="all"
              />
            )}

            <HStack justify="end">
              {isSelf &&
                (isEditing ? (
                  <SaveAction size="sm">{t.actions.save}</SaveAction>
                ) : (
                  <EditAction
                    size="sm"
                    variant="ghost"
                    onClick={handlers.handleSetEditing}
                  >
                    {t.actions.edit}
                  </EditAction>
                ))}
              <MemberOptionsMenu profile={profile} asChild>
                <MoreAction type="button" size="sm" />
              </MemberOptionsMenu>
            </HStack>
          </Flex>

          <HStack gap="1">
            <styled.p color="fg.muted" wordBreak="keep-all">
              {t.members.joined}{" "}
              <styled.time textWrap="nowrap">
                {formatDistanceToNow(new Date(profile.createdAt), {
                  addSuffix: true,
                  locale: dateLocale,
                })}
              </styled.time>
            </styled.p>
            <DotSeparator />
            <HStack
              gap="1"
              color="fg.subtle"
              wordBreak="keep-all"
              textWrap="nowrap"
            >
              <Box flexShrink="0">
                <LikeIcon w="4" />
              </Box>
              <span>
                {profile.like_score}{" "}
                {profile.like_score === 1 ? t.thread.like : t.thread.likes}
              </span>
            </HStack>

            {!isEditing && semester !== undefined && (
              <>
                <DotSeparator />
                <styled.p color="fg.subtle" textWrap="nowrap">
                  {formatSemester(semester)}
                </styled.p>
              </>
            )}
          </HStack>

          {isEditing && (
            <LStack gap="1" maxW={{ base: "full", sm: "64" }}>
              <styled.label
                htmlFor="profile-semester"
                color="fg.muted"
                fontSize="sm"
              >
                {t.profile.semester}
              </styled.label>
              <styled.select
                id="profile-semester"
                width="full"
                height="9"
                px="2"
                borderWidth="thin"
                borderStyle="solid"
                borderColor="border.default"
                borderRadius="l2"
                bg="bg.default"
                color="fg.default"
                {...form.register("semester")}
              >
                <option value="">{t.profile.semesterUnset}</option>
                {SEMESTER_OPTIONS.map((value) => (
                  <option key={value} value={value}>
                    {formatSemester(value)}
                  </option>
                ))}
              </styled.select>
            </LStack>
          )}

          {isEmpty && !isEditing ? (
            <styled.p color="fg.subtle" fontStyle="italic">
              {t.profile.noBio}
            </styled.p>
          ) : (
            <ContentFormField<Form>
              control={form.control}
              name="bio"
              initialValue={profile.bio}
              disabled={!isEditing}
              placeholder={t.profile.noBio}
            />
          )}

          {signaturesEnabled &&
            (isSignatureEmpty && !isEditing ? (
              <styled.p color="fg.subtle" fontStyle="italic">
                {t.profile.noSignature}
              </styled.p>
            ) : (
              <ContentFormField<Form>
                control={form.control}
                name="signature"
                initialValue={profile.signature ?? ""}
                disabled={!isEditing}
                placeholder={t.profile.noSignature}
              />
            ))}
        </styled.form>

        {profile.deletedAt && (
          <Box p="3">
            <ProfileSuspendedBanner date={new Date(profile.deletedAt)} />
          </Box>
        )}

        {canViewAccount && (
          <Box p="3">
            <ProfileAccountManagement accountId={profile.id} />
          </Box>
        )}
      </CardBox>

      <ProfileContent session={session} profile={profile} />
    </LStack>
  );
}
