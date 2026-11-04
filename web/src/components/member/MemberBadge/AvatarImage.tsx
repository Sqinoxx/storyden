"use client";

import Image from "next/image";
import { useState } from "react";

import { css } from "@/styled-system/css";
import { Box } from "@/styled-system/jsx";

const MAX_RETRIES = 2;
const RETRY_DELAY_MS = 750;

const avatarStyles = css({
  borderRadius: "full",
});

type Props = {
  src: string;
  alt: string;
  width: number;
  height: number;
  eager?: boolean;
};

export function AvatarImage({ src, alt, width, height, eager }: Props) {
  const [attempt, setAttempt] = useState(0);
  const [failed, setFailed] = useState(false);

  function handleError() {
    if (attempt >= MAX_RETRIES) {
      setFailed(true);
      return;
    }

    setTimeout(() => setAttempt((a) => a + 1), RETRY_DELAY_MS * (attempt + 1));
  }

  if (failed) {
    return (
      <Box
        role="img"
        aria-label={alt}
        borderRadius="full"
        bgColor="bg.muted"
        style={{ width, height }}
      />
    );
  }

  const resolvedSrc =
    attempt === 0
      ? src
      : `${src}${src.includes("?") ? "&" : "?"}retry=${attempt}`;

  return (
    <Image
      key={resolvedSrc}
      className={avatarStyles}
      src={resolvedSrc}
      alt={alt}
      width={width}
      height={height}
      loading={eager ? "eager" : "lazy"}
      onError={handleError}
    />
  );
}
