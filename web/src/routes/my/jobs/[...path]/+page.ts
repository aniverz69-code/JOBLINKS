import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

// The Tracking section moved from /my/jobs → /my/tracking. Permanently redirect the old
// URLs (index and every sub-path: pipeline/history/…) so bookmarks and inbound links land
// on the canonical path. The rest param matches the empty path too, so this one route
// covers /my/jobs itself. Query string is preserved.
export const load: PageLoad = ({ params, url }) => {
  const rest = params.path ? `/${params.path}` : '';
  redirect(308, `/my/tracking${rest}${url.search}`);
};
