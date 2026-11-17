import { Box, styled } from "@/styled-system/jsx";

type Props = {
  isError: boolean;
  message: string;
  hint?: boolean;
};

export function ContentDragOverlay({ isError, message, hint }: Props) {
  return (
    <Box
      position="absolute"
      top="0"
      left="0"
      right="0"
      bottom="0"
      zIndex="docked"
      pointerEvents="none"
      display="flex"
      alignItems="center"
      justifyContent="center"
      backgroundColor={hint ? "accent.default/10" : "bg.emphasized"}
      borderWidth="medium"
      borderStyle="dashed"
      borderColor={isError ? "border.error" : "accent.default"}
      borderRadius="md"
      style={hint ? undefined : { opacity: 0.95 }}
      role="status"
      aria-live="polite"
      aria-label={message}
      data-drop-hint={hint ? "true" : undefined}
    >
      <styled.div
        fontSize={hint ? "xs" : "sm"}
        fontWeight="medium"
        color={isError ? "fg.error" : "accent.default"}
        bg={hint ? "bg.default" : undefined}
        px={hint ? "2" : undefined}
        py={hint ? "0.5" : undefined}
        borderRadius="sm"
        display="flex"
        flexDirection="column"
        alignItems="center"
        gap="2"
      >
        <span>{message}</span>
      </styled.div>
    </Box>
  );
}
