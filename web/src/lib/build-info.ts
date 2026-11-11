export type BuildInfo = {
  number: string;
  commit?: string;
  date?: string;
};

export function formatBuildInfo(info: BuildInfo | undefined): string | undefined {
  if (!info) {
    return undefined;
  }

  return ["Build " + info.number, info.commit, info.date]
    .filter(Boolean)
    .join(" · ");
}

export function readBuildInfo(): BuildInfo | undefined {
  const number = process.env["NEXT_PUBLIC_BUILD_NUMBER"];
  if (!number) {
    return undefined;
  }

  return {
    number,
    commit: process.env["NEXT_PUBLIC_BUILD_COMMIT"] || undefined,
    date: process.env["NEXT_PUBLIC_BUILD_DATE"] || undefined,
  };
}
