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
const TAP_ZOOM_SCALE = 2.5;
const DOUBLE_TAP_MS = 280;
const SWIPE_DISTANCE = 80;
const DISMISS_DISTANCE = 120;
const AXIS_LOCK_DISTANCE = 8;
const SLIDE_MS = 320;
const SLIDE_EASING = "cubic-bezier(0.22, 1, 0.36, 1)";
const SLIDE_GAP = "var(--spacing-6)";

type Transform = { scale: number; x: number; y: number };
type Offset = { x: number; y: number };
type Axis = "x" | "y" | null;

const IDENTITY: Transform = { scale: 1, x: 0, y: 0 };
const ORIGIN: Offset = { x: 0, y: 0 };

const clamp = (v: number, min: number, max: number) =>
  Math.min(Math.max(v, min), max);

const wrap = (v: number, n: number) => ((v % n) + n) % n;

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
  const thumbsRef = useRef<HTMLDivElement>(null);
  const lastTap = useRef(0);
  const tapTimer = useRef<number | undefined>(undefined);
  const slideTimer = useRef<number | undefined>(undefined);
  const busy = useRef(false);
  const axis = useRef<Axis>(null);

  const count = assets.length;
  const hasMultiple = count > 1;

  const [virtual, setVirtual] = useState(index);
  const current = wrap(virtual, count);

  const [transform, setTransformState] = useState<Transform>(IDENTITY);
  const transformRef = useRef<Transform>(IDENTITY);
  const [smoothZoom, setSmoothZoom] = useState(false);

  const [drag, setDrag] = useState<Offset>(ORIGIN);
  const [shift, setShift] = useState(0);
  const [sliding, setSliding] = useState(false);
  const [chrome, setChrome] = useState(true);

  const zoomed = transform.scale > MIN_SCALE;

  const setTransform = useCallback((next: Transform, animate = false) => {
    transformRef.current = next;
    setSmoothZoom(animate);
    setTransformState(next);
  }, []);

  useEffect(() => {
    if (current !== index) onIndexChange(current);
  }, [current, index, onIndexChange]);

  useEffect(
    () => () => {
      window.clearTimeout(tapTimer.current);
      window.clearTimeout(slideTimer.current);
    },
    [],
  );

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
      zoomAt(
        rect.left + rect.width / 2,
        rect.top + rect.height / 2,
        scale,
        true,
      );
    },
    [zoomAt],
  );

  const resetZoom = useCallback(
    () => setTransform(IDENTITY, true),
    [setTransform],
  );

  const settle = useCallback((after?: () => void) => {
    busy.current = true;
    setSliding(true);
    window.clearTimeout(slideTimer.current);
    slideTimer.current = window.setTimeout(() => {
      after?.();
      setSliding(false);
      busy.current = false;
    }, SLIDE_MS);
  }, []);

  const snapBack = useCallback(() => {
    setDrag(ORIGIN);
    settle();
  }, [settle]);

  const goTo = useCallback(
    (direction: 1 | -1) => {
      if (!hasMultiple || busy.current) return;

      setDrag(ORIGIN);
      setShift(direction);
      settle(() => {
        setShift(0);
        setVirtual((v) => v + direction);
        setTransform(IDENTITY);
      });
    },
    [hasMultiple, setTransform, settle],
  );

  const jumpTo = useCallback(
    (target: number) => {
      if (target === current || busy.current) return;
      setVirtual((v) => v + (target - current));
      setTransform(IDENTITY);
    },
    [current, setTransform],
  );

  useEffect(() => {
    const strip = thumbsRef.current;
    const thumb = strip?.children[current] as HTMLElement | undefined;
    if (!strip || !thumb) return;

    strip.scrollTo({
      left: thumb.offsetLeft - (strip.clientWidth - thumb.offsetWidth) / 2,
      behavior: "smooth",
    });
  }, [current]);

  useEffect(() => {
    const html = document.documentElement.style;
    const body = document.body.style;
    const saved = {
      htmlOverflow: html.overflow,
      htmlOverscroll: html.overscrollBehavior,
      bodyOverflow: body.overflow,
    };

    html.overflow = "hidden";
    html.overscrollBehavior = "none";
    body.overflow = "hidden";
    closeRef.current?.focus({ preventScroll: true });

    return () => {
      html.overflow = saved.htmlOverflow;
      html.overscrollBehavior = saved.htmlOverscroll;
      body.overflow = saved.bodyOverflow;
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

  function handleTap(event: Event, cx: number, cy: number) {
    const onImage = event.target === imageRef.current;
    const isMouse =
      "pointerType" in event && (event as PointerEvent).pointerType === "mouse";

    if (isMouse) {
      if (!onImage) {
        onClose();
      } else if (transformRef.current.scale > MIN_SCALE) {
        resetZoom();
      } else {
        zoomAt(cx, cy, TAP_ZOOM_SCALE, true);
      }
      return;
    }

    const now = Date.now();
    if (now - lastTap.current < DOUBLE_TAP_MS) {
      lastTap.current = 0;
      window.clearTimeout(tapTimer.current);
      if (transformRef.current.scale > MIN_SCALE) {
        resetZoom();
      } else if (onImage) {
        zoomAt(cx, cy, TAP_ZOOM_SCALE, true);
      }
      return;
    }

    lastTap.current = now;
    window.clearTimeout(tapTimer.current);
    tapTimer.current = window.setTimeout(
      () => setChrome((c) => !c),
      DOUBLE_TAP_MS,
    );
  }

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
        first,
        last,
        xy: [cx, cy],
        delta: [dx, dy],
        movement: [mx, my],
        swipe: [swipeX, swipeY],
        pinching,
        cancel,
      }) => {
        if (pinching || busy.current) {
          cancel();
          return;
        }

        if (tap) {
          handleTap(event, cx, cy);
          return;
        }

        const prev = transformRef.current;
        if (prev.scale > MIN_SCALE) {
          setTransform(
            bounded({ scale: prev.scale, x: prev.x + dx, y: prev.y + dy }),
          );
          return;
        }

        if (first) axis.current = null;
        if (
          !axis.current &&
          Math.max(Math.abs(mx), Math.abs(my)) > AXIS_LOCK_DISTANCE
        ) {
          axis.current = Math.abs(mx) >= Math.abs(my) ? "x" : "y";
        }

        if (!last) {
          if (axis.current === "x") {
            setDrag({ x: hasMultiple ? mx : mx * 0.2, y: 0 });
          } else if (axis.current === "y") {
            setDrag({ x: 0, y: Math.max(my, 0) });
          }
          return;
        }

        const locked = axis.current;
        axis.current = null;

        if (locked === "x" && hasMultiple) {
          if (mx <= -SWIPE_DISTANCE || swipeX === -1) {
            goTo(1);
            return;
          }
          if (mx >= SWIPE_DISTANCE || swipeX === 1) {
            goTo(-1);
            return;
          }
        }

        if (locked === "y" && (my >= DISMISS_DISTANCE || swipeY === 1)) {
          onClose();
          return;
        }

        snapBack();
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

  const active = assets[current];
  if (!active) return null;

  const dismissProgress = clamp(drag.y / 400, 0, 0.6);
  const slides = hasMultiple ? [-1, 0, 1] : [0];

  return (
    <Portal>
      <Box
        role="dialog"
        aria-modal="true"
        aria-label={title}
        position="fixed"
        inset="0"
        zIndex="overlay"
        display="grid"
        gridTemplateRows="auto 1fr auto"
        overflow="hidden"
        touchAction="none"
        overscrollBehavior="none"
        color="white"
        bg="black"
        style={{
          backgroundColor: `rgba(0, 0, 0, ${1 - dismissProgress})`,
        }}
      >
        <styled.img
          key={`ambient-${active.id}`}
          src={getAssetURL(active.path)}
          alt=""
          aria-hidden
          position="absolute"
          inset="0"
          w="full"
          h="full"
          objectFit="cover"
          pointerEvents="none"
          style={{
            filter: "blur(64px) saturate(1.6)",
            transform: "scale(1.25)",
            opacity: 0.35 * (1 - dismissProgress * 1.6),
          }}
        />

        <HStack
          position="relative"
          justify="space-between"
          gap="3"
          px={{ base: "3", md: "5" }}
          pb="2"
          transition="opacity"
          transitionDuration="normal"
          style={{
            paddingTop: "max(var(--spacing-3), env(safe-area-inset-top))",
            opacity: chrome ? 1 - dismissProgress * 1.6 : 0,
            pointerEvents: chrome ? "auto" : "none",
          }}
        >
          <HStack gap="3" minW="0">
            {hasMultiple && (
              <styled.span
                flexShrink="0"
                fontSize="sm"
                fontWeight="semibold"
                fontVariantNumeric="tabular-nums"
                px="3"
                py="1.5"
                borderRadius="full"
                bg="white.a3"
                borderWidth="thin"
                borderColor="white.a4"
                backdropBlur="frosted"
                backdropFilter="auto"
              >
                {current + 1} / {count}
              </styled.span>
            )}
            <styled.span
              fontSize="sm"
              fontWeight="medium"
              color="white.a11"
              truncate
              display={{ base: "none", sm: "block" }}
            >
              {title}
            </styled.span>
          </HStack>

          <HStack gap="2" flexShrink="0">
            <HStack
              gap="0.5"
              p="1"
              borderRadius="full"
              bg="white.a3"
              borderWidth="thin"
              borderColor="white.a4"
              backdropBlur="frosted"
              backdropFilter="auto"
              display={{ base: "none", md: "flex" }}
            >
              <ControlButton
                variant="ghost"
                onClick={() => zoomCentre(transform.scale / 1.5)}
                disabled={!zoomed}
                aria-label={labels.zoomOut}
                title={labels.zoomOut}
              >
                <ZoomOut />
              </ControlButton>
              <styled.button
                type="button"
                onClick={resetZoom}
                disabled={!zoomed}
                aria-label={labels.resetZoom}
                title={labels.resetZoom}
                minW="12"
                fontSize="xs"
                fontWeight="semibold"
                fontVariantNumeric="tabular-nums"
                color="white.a11"
                cursor={zoomed ? "pointer" : "default"}
              >
                {Math.round(transform.scale * 100)}%
              </styled.button>
              <ControlButton
                variant="ghost"
                onClick={() => zoomCentre(transform.scale * 1.5)}
                disabled={transform.scale >= MAX_SCALE}
                aria-label={labels.zoomIn}
                title={labels.zoomIn}
              >
                <ZoomIn />
              </ControlButton>
            </HStack>
            {zoomed && (
              <ControlButton
                onClick={resetZoom}
                aria-label={labels.resetZoom}
                title={labels.resetZoom}
                display={{ base: "flex", md: "none" }}
              >
                <RotateCcw />
              </ControlButton>
            )}
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

        <Box position="relative" minH="0">
          <Box
            ref={stageRef}
            position="absolute"
            inset="0"
            overflow="hidden"
            touchAction="none"
            userSelect="none"
          >
            <Box
              position="absolute"
              inset="0"
              style={{
                transform: `translate3d(calc(${-shift} * (100% + ${SLIDE_GAP}) + ${drag.x}px), ${drag.y}px, 0)`,
                transition: sliding
                  ? `transform ${SLIDE_MS}ms ${SLIDE_EASING}`
                  : undefined,
              }}
            >
              {slides.map((offset) => {
                const position = virtual + offset;
                const asset = assets[wrap(position, count)];
                if (!asset) return null;
                const isCurrent = offset === 0;

                return (
                  <Box
                    key={position}
                    position="absolute"
                    inset="0"
                    display="flex"
                    alignItems="center"
                    justifyContent="center"
                    px={{ base: "0", md: "20" }}
                    py={{ base: "0", md: "2" }}
                    aria-hidden={!isCurrent}
                    style={{
                      transform: `translateX(calc(${offset} * (100% + ${SLIDE_GAP})))`,
                    }}
                  >
                    <styled.img
                      ref={isCurrent ? imageRef : undefined}
                      src={getAssetURL(asset.path)}
                      alt={`${title} ${wrap(position, count) + 1}`}
                      draggable={false}
                      decoding="async"
                      maxW="full"
                      maxH="full"
                      objectFit="contain"
                      userSelect="none"
                      borderRadius={{ base: "none", md: "md" }}
                      shadow="2xl"
                      style={{
                        cursor: isCurrent
                          ? zoomed
                            ? "grab"
                            : "zoom-in"
                          : undefined,
                        transform: isCurrent
                          ? `translate3d(${transform.x}px, ${transform.y}px, 0) scale(${transform.scale})`
                          : undefined,
                        transition:
                          isCurrent && smoothZoom
                            ? "transform 0.22s ease-out"
                            : undefined,
                        willChange: isCurrent ? "transform" : undefined,
                      }}
                    />
                  </Box>
                );
              })}
            </Box>
          </Box>

          {hasMultiple && (
            <>
              <NavZone left="0">
                <ControlButton
                  size="lg"
                  onClick={() => goTo(-1)}
                  aria-label={labels.previous}
                  title={labels.previous}
                  style={
                    chrome && !zoomed
                      ? undefined
                      : { opacity: 0, pointerEvents: "none" }
                  }
                >
                  <ChevronLeft />
                </ControlButton>
              </NavZone>
              <NavZone right="0">
                <ControlButton
                  size="lg"
                  onClick={() => goTo(1)}
                  aria-label={labels.next}
                  title={labels.next}
                  style={
                    chrome && !zoomed
                      ? undefined
                      : { opacity: 0, pointerEvents: "none" }
                  }
                >
                  <ChevronRight />
                </ControlButton>
              </NavZone>
            </>
          )}
        </Box>

        {hasMultiple ? (
          <Box
            position="relative"
            transition="opacity"
            transitionDuration="normal"
            style={{
              paddingBottom:
                "max(var(--spacing-3), env(safe-area-inset-bottom))",
              opacity: chrome ? 1 - dismissProgress * 1.6 : 0,
              pointerEvents: chrome ? "auto" : "none",
            }}
          >
            <HStack
              ref={thumbsRef}
              gap="2"
              px="4"
              pt="2"
              pb="1"
              w="fit"
              maxW="full"
              mx="auto"
              position="relative"
              overflowX="auto"
              overscrollBehaviorX="contain"
              touchAction="pan-x"
              scrollbar="hidden"
            >
              {assets.map((a, i) => (
                <styled.button
                  key={a.id}
                  type="button"
                  onClick={() => jumpTo(i)}
                  aria-label={`${title} ${i + 1}`}
                  aria-current={i === current}
                  flexShrink="0"
                  w={{ base: "12", md: "14" }}
                  h={{ base: "12", md: "14" }}
                  borderRadius="md"
                  overflow="hidden"
                  cursor="pointer"
                  outlineStyle="solid"
                  outlineOffset="0.5"
                  outlineWidth={i === current ? "medium" : "thin"}
                  outlineColor={i === current ? "white" : "transparent"}
                  opacity={i === current ? "full" : "5"}
                  transition="all"
                  transitionDuration="fast"
                  _hover={{ opacity: "full" }}
                >
                  <styled.img
                    src={getAssetURL(a.path)}
                    alt=""
                    draggable={false}
                    w="full"
                    h="full"
                    objectFit="cover"
                  />
                </styled.button>
              ))}
            </HStack>
          </Box>
        ) : (
          <Box
            style={{
              paddingBottom: "env(safe-area-inset-bottom)",
            }}
          />
        )}
      </Box>
    </Portal>
  );
}

const NavZone = styled("div", {
  base: {
    position: "absolute",
    top: "0",
    bottom: "0",
    display: { base: "none", md: "flex" },
    alignItems: "center",
    px: "4",
    pointerEvents: "none",
    "& > button": { pointerEvents: "auto" },
  },
});

const ControlButton = styled(
  "button",
  {
    base: {
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      flexShrink: "0",
      borderRadius: "full",
      color: "white",
      cursor: "pointer",
      transition: "all",
      transitionDuration: "fast",
      _disabled: { opacity: "3", cursor: "default" },
      _focusVisible: {
        outlineWidth: "medium",
        outlineStyle: "solid",
        outlineColor: "white",
      },
    },
    variants: {
      variant: {
        glass: {
          bg: "white.a3",
          borderWidth: "thin",
          borderColor: "white.a4",
          backdropBlur: "frosted",
          backdropFilter: "auto",
          _hover: { bg: "white.a5" },
        },
        ghost: {
          bg: "transparent",
          _hover: { bg: "white.a4" },
        },
      },
      size: {
        md: {
          w: "10",
          h: "10",
          "& svg": { w: "5", h: "5" },
        },
        lg: {
          w: "12",
          h: "12",
          "& svg": { w: "6", h: "6" },
        },
      },
    },
    compoundVariants: [
      { variant: "ghost", size: "md", css: { w: "8", h: "8" } },
      {
        variant: "ghost",
        size: "md",
        css: { "& svg": { w: "4", h: "4" } },
      },
    ],
    defaultVariants: { variant: "glass", size: "md" },
  },
  { defaultProps: { type: "button" } },
);
