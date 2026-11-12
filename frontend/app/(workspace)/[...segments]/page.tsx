import { notFound, redirect } from "next/navigation";
import { loanRoutes, workspaceRoute } from "@/lib/workspace-routes";

export default async function WorkspacePage({ params }: { params: Promise<{ segments: string[] }> }) {
  const { segments } = await params;
  const path = "/" + segments.join("/");
  if (path === "/pinjaman") redirect(loanRoutes.applications);
  if (!workspaceRoute(path)) notFound();
  return null;
}
