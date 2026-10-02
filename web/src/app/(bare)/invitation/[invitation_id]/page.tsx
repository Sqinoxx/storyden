import { invitationGet } from "@/api/openapi-server/invitations";
import { getServerSession } from "@/auth/server-session";
import { MemberIdent } from "@/components/member/MemberBadge/MemberIdent";
import { UnreadyBanner } from "@/components/site/Unready";
import { LinkButton } from "@/components/ui/link-button";
import { Trans } from "@/lib/i18n";
import { getSettings } from "@/lib/settings/settings-server";
import { Box, Divider, LStack, VStack, styled } from "@/styled-system/jsx";

type Props = {
  params: Promise<{
    invitation_id: string;
  }>;
};

export default async function Page({ params }: Props) {
  try {
    const { invitation_id: invitationID } = await params;
    const [{ data: invitation }, settings, session] = await Promise.all([
      invitationGet(invitationID, { cache: "no-store" }),
      getSettings(),
      getServerSession({ cache: "no-store" }),
    ]);

    return (
      <LStack gap="4" textAlign="center" alignItems="center">
        <Divider />

        <VStack gap="1">
          <Box>
            <MemberIdent
              profile={invitation.creator}
              size="md"
              name="full-vertical"
            />
          </Box>
          <styled.p>
            <Trans path="auth.invitedYouTo" /> <strong>{settings.title}</strong>
          </styled.p>
        </VStack>

        {session ? (
          <VStack
            w="full"
            gap="4"
            borderWidth="thin"
            borderStyle="solid"
            borderColor="border.warning"
            bgColor="bg.warning"
            color="fg.warning"
            borderRadius="md"
            p="4"
          >
            <VStack gap="1" textWrap="balance">
              <styled.p fontWeight="semibold">
                <Trans path="auth.alreadySignedInAs" />{" "}
                <strong>{session.handle}</strong> ({settings.title})
              </styled.p>
              <styled.p fontSize="sm">
                <Trans path="auth.cannotAcceptWhileSignedIn" />
              </styled.p>
            </VStack>

            <LinkButton w="full" href="/">
              <Trans path="nav.home" />
            </LinkButton>
          </VStack>
        ) : (
          <LinkButton
            w="full"
            href={`/register?invitation_id=${invitation.id}`}
          >
            <Trans path="auth.acceptInvitation" />
          </LinkButton>
        )}
      </LStack>
    );
  } catch (error) {
    return <UnreadyBanner error={error} />;
  }
}

export async function generateMetadata({ params }: Props) {
  try {
    const { invitation_id: invitationID } = await params;
    const [{ data: invitation }, settings] = await Promise.all([
      invitationGet(invitationID, { cache: "no-store" }),
      getSettings(),
    ]);

    return {
      title: `${invitation.creator.name} invited you to ${settings.title}`,
      description: `Accept your invitation to join ${settings.title}.`,
    };
  } catch {
    return {
      title: "Invitation",
      description: "Accept your invitation to join the community.",
    };
  }
}
