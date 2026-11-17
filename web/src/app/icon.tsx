import { streamMetadataImage } from "@/lib/metadata/image-response";

export const size = {
  width: 512,
  height: 512,
};
export const contentType = "image/png";
export const dynamic = "force-dynamic";

export default async function Icon() {
  return streamMetadataImage("/info/icon/512x512");
}
