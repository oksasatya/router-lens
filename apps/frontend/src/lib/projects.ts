import { queryOptions } from "@tanstack/react-query";
import { getProject, listProjects } from "@/services/projectService";

export function projectsQueryOptions(page: number, limit: number) {
  return queryOptions({
    queryKey: ["projects", { page, limit }],
    queryFn: () => listProjects(page, limit),
  });
}

export function projectQueryOptions(id: string) {
  return queryOptions({
    queryKey: ["projects", id],
    queryFn: () => getProject(id),
  });
}
