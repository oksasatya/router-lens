import { createFileRoute, Outlet } from "@tanstack/react-router";

// Pathless layout: the list lives at the index child (_app.projects.index.tsx),
// the detail view at the $projectId child. This file exists only so both
// children have somewhere to render into — without this Outlet, navigating to
// /projects/$projectId updates the URL but the list route's own JSX (which has
// no Outlet) keeps rendering in its place.
export const Route = createFileRoute("/_app/projects")({
  component: () => <Outlet />,
});
