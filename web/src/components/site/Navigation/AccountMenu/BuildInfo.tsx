import { Box } from "@/styled-system/jsx";
import { formatBuildInfo, readBuildInfo } from "@/lib/build-info";

export function BuildInfo() {
  const label = formatBuildInfo(readBuildInfo());
  if (!label) {
    return null;
  }

  return (
    <Box
      px="3"
      py="1"
      fontSize="xs"
      color="fg.muted"
      userSelect="text"
      data-testid="build-info"
    >
      {label}
    </Box>
  );
}
