import { z } from "zod";

export const MIN_SEMESTER = 1;
export const MAX_SEMESTER = 12;

// Reported by the server in place of a semester number once a member has
// progressed beyond MAX_SEMESTER without setting it manually again.
export const FINISHED_SEMESTER = -1;

export const ACADEMIC_META_KEY = "academic";

export const AcademicMetaSchema = z.object({
  semester: z
    .number()
    .int()
    .refine(
      (value) =>
        value === FINISHED_SEMESTER ||
        (value >= MIN_SEMESTER && value <= MAX_SEMESTER),
    ),
  // Stamped by the server. Clients only ever send the semester number; the
  // term is what lets the value advance on its own each April and October.
  term: z.string().optional(),
});

export type AcademicMeta = z.infer<typeof AcademicMetaSchema>;

export const SEMESTER_OPTIONS = Array.from(
  { length: MAX_SEMESTER - MIN_SEMESTER + 1 },
  (_, i) => i + MIN_SEMESTER,
);

export function parseAcademicMeta(
  meta: Record<string, unknown> | undefined,
): AcademicMeta | undefined {
  const parsed = AcademicMetaSchema.safeParse(meta?.[ACADEMIC_META_KEY]);

  return parsed.success ? parsed.data : undefined;
}

export function formatSemester(semester: number): string {
  if (semester === FINISHED_SEMESTER) {
    return "Fertig";
  }

  return `${semester}. Fachsemester`;
}
