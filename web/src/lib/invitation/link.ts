import { WEB_ADDRESS } from "@/config";

export function createInvitationLink(invitationID: string) {
  const webAddress = WEB_ADDRESS.replace(/\/$/, "");

  return `${webAddress}/invitation/${invitationID}`;
}
