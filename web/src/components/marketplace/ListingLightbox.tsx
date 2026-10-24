"use client";

import { Portal } from "@ark-ui/react";
import { useGesture } from "@use-gesture/react";
import {
  ChevronLeft,
  ChevronRight,
  RotateCcw,
  X,
  ZoomIn,
  ZoomOut,
} from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

import { Asset } from "@/api/openapi-schema";
import { useTranslation } from "@/lib/i18n";
import { Box, HStack, styled } from "@/styled-system/jsx";
import { getAssetURL } from "@/utils/asset";

const MIN_SCALE = 1;
const MAX_SCALE = 5;
const DOUBLE_TAP_SCALE = 2.5;
const DOUBLE_TAP_MS = 300;
const SWIPE_DISTANCE = 80;
const DISMISS_DISTANCE = 120;

type Transform = { scale: number; x: number; y: number };

const IDENTITY: Transform = { scale: 1, x: 0, y: 0 };

const clamp = (v: number, min: number, max: number) =>
  Math.min(Math.max(v, min), max);

type Props = {
  assets: Asset[];
  index: number;
  title: string;
  onIndexChange: (index: number) => void;
  onClose: () => void;
};

export function ListingLightbox({
  assets,
  index,
  title,
  onIndexChange,
  onClose,
}: Props) {
  const t = useTranslation();
  const labels = t.marketplace.lightbox;

  const stageRef = useRef<HTMLDivElement>(null);
  const imageRef = useRef<HTMLImageElement>(null);
  const closeRef = useRef<HTMLButtonElement>(null);
  const lastTap = useRef(0);

  const [transform, setTransformState] = useState<Transform>(IDENTITY);
  const transformRef = useRef<Transform>(IDENTITY);
  const [smooth, setSmooth] = useState(false);

  const count = assets.length;
  const current = assets[index] ?? assets[0];
  const hasMultiple = count > 1;

  const setTransform = useCallback((next: Transform, animate = false) => {
    transformRef.current = next;
    setSmooth(animate);
    setTransformState(next);
  }, []);

  const bounded = useCallback((next: Transform): Transform => {
    const stage = stageRef.current;
    const image = imageRef.current;
    if (!stage || !image || next.scale <= MIN_SCALE) {
      return { scale: Math.max(next.scale, MIN_SCALE), x: 0, y: 0 };
    }

    const maxX = Math.max(
      0,
      (image.offsetWidth * next.scale - stage.clientWidth) / 2,
    );
    const maxY = Math.max(
      0,
      (image.offsetHeight * next.scale - stage.clientHeight) / 2,
    );

    return {
      scale: next.scale,
      x: clamp(next.x, -maxX, maxX),
      y: clamp(next.y, -maxY, maxY),
    };
  }, []);

  const zoomAt = useCallback(
    (clientX: number, clientY: number, scale: number, animate = false) => {
      const stage = stageRef.current;
      if (!stage) return;

      const rect = stage.getBoundingClientRect();
      const px = clientX - (rect.left + rect.width / 2);
      const py = clientY - (rect.top + rect.height / 2);

      const prev = transformRef.current;
      const nextScale = clamp(scale, MIN_SCALE, MAX_SCALE);
      const ratio = nextScale / prev.scale;

      setTransform(
        bounded({
          scale: nextScale,
          x: px - (px - prev.x) * ratio,
          y: py - (py - prev.y) * ratio,
        }),
        animate,
      );
    },
    [bounded, setTransform],
  );

  const zoomCentre = useCallback(
    (scale: number) => {
      const rect = stageRef.current?.getBoundingClientRect();
      if (!rect) return;
      zoomAt(rect.left + rect.width / 2, rect.top + rect.height / 2, scale, true);
    },
    [zoomAt],
  );

  const resetZoom = useCallback(
    () => setTransform(IDENTITY, true),
    [setTransform],
  );

  const goTo = useCallback(
    (offset: number) => {
      if (!hasMultiple) return;
      onIndexChange((index + offset + count) % count);
    },
    [count, hasMultiple, index, onIndexChange],
  );

  useEffect(() => {
    setTransform(IDENTITY);
  }, [index, setTransform]);

  useEffect(() => {
    const { overflow } = document.body.style;
    document.body.style.overflow = "hidden";
    closeRef.current?.focus();
    return () => {
      document.body.style.overflow = overflow;
    };
  }, []);

  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      switch (e.key) {
        case "Escape":
          onClose();
          break;
        case "ArrowLeft":
          goTo(-1);
          break;
        case "ArrowRight":
          goTo(1);
          break;
        case "+":
        case "=":
          zoomCentre(transformRef.current.scale * 1.5);
          break;
        case "-":
          zoomCentre(transformRef.current.scale / 1.5);
          break;
        case "0":
          resetZoom();
          break;
        default:
          return;
      }
      e.preventDefault();
    }

    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [goTo, onClose, resetZoom, zoomCentre]);

  useGesture(
    {
      onPinch: ({ offset: [scale], origin: [ox, oy], event }) => {
        event.preventDefault();
        zoomAt(ox, oy, scale);
      },
      onWheel: ({ event, delta: [, dy] }) => {
        if (event.ctrlKey) return;
        event.preventDefault();
        zoomAt(
          event.clientX,
          event.clientY,
          transformRef.current.scale * Math.exp(-dy * 0.002),
        );
      },
      onDrag: ({
        tap,
        event,
        xy: [cx, cy],
        delta: [dx, dy],
        movement: [mx, my],
        swipe: [swipeX, swipeY],
        pinching,
        last,
        cancel,
      }) => {
        if (pinching) {
          cancel();
          return;
        }

        if (tap) {
          const now = Date.now();
          const onImage = event.target === imageRef.current;

          if (onImage && now - lastTap.current < DOUBLE_TAP_MS) {
            lastTap.current = 0;
            if (transformRef.current.scale > MIN_SCALE) {
              resetZoom();
            } else {
              zoomAt(cx, cy, DOUBLE_TAP_SCALE, true);
            }
            return;
          }

          lastTap.current = now;
          if (!onImage && transformRef.current.scale <= MIN_SCALE) {
            onClose();
          }
          return;
        }

        const prev = transformRef.current;

        if (prev.scale > MIN_SCALE) {
          setTransform(
            bounded({ scale: prev.scale, x: prev.x + dx, y: prev.y + dy }),
          );
          return;
        }

        const horizontal = Math.abs(mx) >= Math.abs(my);

        if (!last) {
          setTransform({
            scale: 1,
            x: horizontal && hasMultiple ? mx : 0,
            y: horizontal ? 0 : Math.max(my, 0),
          });
          return;
        }

        if (horizontal && hasMultiple) {
          if (mx <= -SWIPE_DISTANCE || swipeX === -1) {
            goTo(1);
            return;
          }
          if (mx >= SWIPE_DISTANCE || swipeX === 1) {
            goTo(-1);
            return;
          }
        }

        if (!horizontal && (my >= DISMISS_DISTANCE || swipeY === 1)) {
          onClose();
          return;
        }

        resetZoom();
      },
    },
    {
      target: stageRef,
      eventOptions: { passive: false },
      drag: { filterTaps: true, pointer: { touch: true } },
      pinch: {
        scaleBounds: { min: MIN_SCALE, max: MAX_SCALE },
        from: () => [transformRef.current.scale, 0],
      },
    },
  );

  if (!current) return null;

  const dismissProgress =
    transform.scale <= MIN_SCALE ? clamp(transform.y / 400, 0, 0.6) : 0;

  return (
    <Portal>
      <Box
        role="dialog"
        aria-modal="true"
        aria-label={title}
        position="fixed"
        inset="0"
        w="screen"
        h="dvh"
        zIndex="overlay"
        bg="black.a12"
        color="white"
        style={{ opacity: 1 - dismissProgress }}
      >
        <Box
          ref={stageRef}
          position="absolute"
          inset="0"
          px={{ base: "2", md: "14" }}
          py="12"
          display="flex"
          alignItems="center"
          justifyContent="center"
          overflow="hidden"
          touchAction="none"
          userSelect="none"
          cursor={transform.scale > MIN_SCALE ? "grab" : "zoom-in"}
        >
          <styled.img
            ref={imageRef}
            key={current.id}
            src={getAssetURL(current.path)}
            alt={`${title} ${index + 1}`}
            draggable={false}
            maxW="full"
            maxH="full"
            objectFit="contain"
            userSelect="none"
            pointerEvents="auto"
            style={{
              transform: `translate3d(${transform.x}px, ${transform.y}px, 0) scale(${transform.scale})`,
              transition: smooth ? "transform 0.2s ease-out" : undefined,
              willChange: "transform",
            }}
          />
        </Box>

        <HStack
          position="absolute"
          top="0"
          left="0"
          right="0"
          justify="space-between"
          gap="1.5"
          p="2"
          pointerEvents="none"
          style={{ paddingTop: "max(var(--spacing-2), env(safe-area-inset-top))" }}
        >
          <styled.span
            fontSize="sm"
            fontWeight="medium"
            px="3"
            py="1"
            borderRadius="full"
            bg="black.a8"
            visibility={hasMultiple ? "visible" : "hidden"}
          >
            {index + 1} / {count}
          </styled.span>

          <HStack gap="2" pointerEvents="auto">
            <ControlButton
              onClick={() => zoomCentre(transform.scale / 1.5)}
              disabled={transform.scale <= MIN_SCALE}
              aria-label={labels.zoomOut}
              title={labels.zoomOut}
            >
              <ZoomOut />
            </ControlButton>
            <ControlButton
              onClick={() => zoomCentre(transform.scale * 1.5)}
              disabled={transform.scale >= MAX_SCALE}
              aria-label={labels.zoomIn}
              title={labels.zoomIn}
            >
              <ZoomIn />
            </ControlButton>
            <ControlButton
              onClick={resetZoom}
              disabled={transform.scale <= MIN_SCALE}
              aria-label={labels.resetZoom}
              title={labels.resetZoom}
              display={{ base: "none", sm: "flex" }}
            >
              <RotateCcw />
            </ControlButton>
            <ControlButton
              ref={closeRef}
              onClick={onClose}
              aria-label={labels.close}
              title={labels.close}
            >
              <X />
            </ControlButton>
          </HStack>
        </HStack>

        {hasMultiple && (
          <>
            <ControlButton
              position="absolute"
              left="2"
              top="1/2"
              transform="translateY(-50%)"
              display={{ base: "none", md: "flex" }}
              onClick={() => goTo(-1)}
              aria-label={labels.previous}
              title={labels.previous}
            >
              <ChevronLeft />
            </ControlButton>
            <ControlButton
              position="absolute"
              right="2"
              top="1/2"
              transform="translateY(-50%)"
              display={{ base: "none", md: "flex" }}
              onClick={() => goTo(1)}
              aria-label={labels.next}
              title={labels.next}
            >
              <ChevronRight />
            </ControlButton>
          </>
        )}
      </Box>
    </Portal>
  );
}

const ControlButton = styled(
  "button",
  {
    base: {
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      w: { base: "8", md: "9" },
      h: { base: "8", md: "9" },
      flexShrink: "0",
      borderRadius: "full",
      bg: "black.a8",
      color: "white",
      cursor: "pointer",
      _hover: { bg: "black.a10" },
      _disabled: { opacity: "4", cursor: "default" },
      _focusVisible: {
        outlineWidth: "medium",
        outlineStyle: "solid",
        outlineColor: "white",
      },
      "& svg": { w: "4", h: "4" },
    },
  },
  { defaultProps: { type: "button" } },
);
