import { redirect } from "next/navigation";
import { menuRoutes } from "@/lib/workspace-routes";

export default function Home() {
  redirect(menuRoutes.members);
}
