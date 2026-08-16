import { RequestError } from "./common";

/**
 * Problem type URNs the client reacts to specifically. The backend derives
 * these from its error kinds, so they must match
 * `urn:storyden:problem:<kebab-cased-kind>`.
 */
export const ProblemType = {
  EmailNotVerified: "urn:storyden:problem:email-not-verified",
  PermissionDenied: "urn:storyden:problem:permission-denied",
} as const;

export type ProblemTypeValue =
  (typeof ProblemType)[keyof typeof ProblemType];

export function isProblemType(error: unknown, type: ProblemTypeValue): boolean {
  return error instanceof RequestError && error.problem?.type === type;
}
