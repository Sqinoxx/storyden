import { FileKind, getFileTypeInfo } from "./fileType";

type Palette = { accent: string; ink: string };

const WHITE = "var(--colors-white)";

const PALETTES: Record<FileKind, Palette> = {
  pdf: { accent: "var(--colors-red-10)", ink: WHITE },
  word: { accent: "var(--colors-blue-10)", ink: WHITE },
  excel: { accent: "var(--colors-green-10)", ink: WHITE },
  csv: { accent: "var(--colors-green-light-11)", ink: WHITE },
  powerpoint: { accent: "var(--colors-orange-10)", ink: WHITE },
  archive: { accent: "var(--colors-amber-9)", ink: "var(--colors-amber-light-12)" },
  image: { accent: "var(--colors-pink-10)", ink: WHITE },
  video: { accent: "#8e4ec6", ink: WHITE },
  audio: { accent: "#12a594", ink: WHITE },
  text: { accent: "var(--colors-slate-10)", ink: WHITE },
  code: { accent: "var(--colors-slate-light-11)", ink: WHITE },
  file: { accent: "var(--colors-gray-9)", ink: WHITE },
};

type Props = {
  mimeType?: string;
  filename?: string;
  size?: number;
  title?: string;
};

export function FileTypeIcon({ mimeType, filename, size = 28, title }: Props) {
  const { kind, label } = getFileTypeInfo(mimeType, filename);
  const { accent, ink } = PALETTES[kind];
  const tint = `color-mix(in srgb, ${accent} 14%, var(--colors-bg-default))`;

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      role={title ? "img" : undefined}
      aria-hidden={title ? undefined : true}
      data-file-kind={kind}
      style={{ flexShrink: 0, display: "block" }}
    >
      {title && <title>{title}</title>}
      <path
        d="M8.5 2.75h11.25l7.5 7.5V27.5a1.75 1.75 0 0 1-1.75 1.75H8.5a1.75 1.75 0 0 1-1.75-1.75v-23A1.75 1.75 0 0 1 8.5 2.75Z"
        fill={tint}
        stroke={accent}
        strokeWidth="1.5"
        strokeLinejoin="round"
      />
      <path
        d="M19.75 2.75v5.75c0 .97.78 1.75 1.75 1.75h5.75"
        fill="none"
        stroke={accent}
        strokeWidth="1.5"
        strokeLinejoin="round"
      />
      {label ? (
        <>
          <rect x="2" y="14.5" width="22" height="10.5" rx="2.25" fill={accent} />
          <text
            x="13"
            y="22.4"
            textAnchor="middle"
            fill={ink}
            fontSize="7.5"
            fontWeight="800"
            fontFamily="ui-sans-serif, system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
            letterSpacing="0.2"
            {...(label.length >= 4
              ? { textLength: 18.5, lengthAdjust: "spacingAndGlyphs" }
              : {})}
          >
            {label}
          </text>
        </>
      ) : (
        <g stroke={accent} strokeWidth="1.5" strokeLinecap="round">
          <path d="M11 15.5h10" />
          <path d="M11 19.5h10" />
          <path d="M11 23.5h6" />
        </g>
      )}
    </svg>
  );
}
