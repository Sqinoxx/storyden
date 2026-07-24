"use client";

import { useDisclosure } from "@/utils/useDisclosure";

import { Node } from "@/api/openapi-schema";
import * as Menu from "@/components/ui/menu";
import { useTranslation } from "@/lib/i18n";

import { ReportIcon } from "../ui/icons/Report";

import { ReportNodeModal } from "./ReportNodeModal";

type Props = {
  node: Node;
};

export function ReportNodeMenuItem({ node }: Props) {
  const disclosure = useDisclosure();
  const t = useTranslation();

  return (
    <>
      <Menu.Item value="report-node" onClick={disclosure.onOpen}>
        <ReportIcon />
        &nbsp; {t.library.reportPage}
      </Menu.Item>

      <ReportNodeModal node={node} {...disclosure} />
    </>
  );
}
