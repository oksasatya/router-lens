import { api } from "@/lib/api";
import { paginated, projectSchema, type Project } from "@/lib/schemas";

export interface ProjectInput {
  name: string;
  description: string;
}

const listSchema = paginated(projectSchema);

/** GET /projects — offset-paginated. */
export async function listProjects(page: number, limit: number) {
  const res = await api.get("/projects", { params: { page, limit } });
  return listSchema.parse(res.data);
}

/** GET /projects/:id */
export async function getProject(id: string): Promise<Project> {
  const res = await api.get(`/projects/${id}`);
  return projectSchema.parse(res.data);
}

/** POST /projects */
export async function createProject(input: ProjectInput): Promise<Project> {
  const res = await api.post("/projects", input);
  return projectSchema.parse(res.data);
}

/** PUT /projects/:id */
export async function updateProject(id: string, input: ProjectInput): Promise<Project> {
  const res = await api.put(`/projects/${id}`, input);
  return projectSchema.parse(res.data);
}

/** DELETE /projects/:id */
export async function deleteProject(id: string): Promise<void> {
  await api.delete(`/projects/${id}`);
}
