import { RequestError, parseProblemDetails } from "@/api/common";
import { ProblemType } from "@/api/problems";

const ErrUnexpected = "An unexpected error occurred";

const ErrAccessDenied =
  "Du hast keinen Zugriff auf diese Seite. Bitte verifiziere dich zuerst.";

// Next.js replaces the real message of any Error thrown during a Server
// Component render with this boilerplate before it reaches the client in
// production builds, so the original reason (e.g. a permission problem) is
// unrecoverable here - only its presence can be detected.
function isRedactedServerComponentError(message: string): boolean {
  return message.includes("Server Components render");
}

function isAccessProblem(type: string | undefined): boolean {
  return (
    type === ProblemType.EmailNotVerified ||
    type === ProblemType.PermissionDenied
  );
}

/**
 * Derives an end-user message from an error/exception value.
 * @param e exception or any error type
 */
export function deriveError(e: unknown): string {
  if (e === null || e === undefined) {
    return "";
  }

  if (typeof e === "string") {
    return e;
  }

  if (e instanceof Error) {
    if (e instanceof RequestError) {
      if (isAccessProblem(e.problem?.type)) {
        return ErrAccessDenied;
      }

      return e.problem?.title ?? e.message;
    }

    if (e instanceof TypeError) {
      console.error(e);
      return ErrUnexpected;
    }

    if (isRedactedServerComponentError(e.message)) {
      return ErrAccessDenied;
    }

    if (e.message.includes("React")) {
      // React prints these by default.
      return "Something went wrong while rendering data.";
    }

    return e.message ?? ErrUnexpected;
  }

  const problem = parseProblemDetails(e);
  if (problem) {
    if (isAccessProblem(problem.type)) {
      return ErrAccessDenied;
    }

    return problem.title ?? ErrUnexpected;
  }

  console.error("unable to derive error text:", e);
  return "An unknown error occurred";
}
