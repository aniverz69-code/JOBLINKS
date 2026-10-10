import { redirect } from '@sveltejs/kit';

// The MCP server lives in the CLI page's `#mcp` section rather than on a page of its own —
// one API key drives both surfaces, so splitting them would duplicate the setup story.
// This route exists because directory listings (the MCP registry, mcp.so, PulseMCP) are
// filled in by hand and several of them strip the fragment, leaving a bare /mcp that used
// to 404. Permanent, so the anchor is what gets indexed.
export const load = () => {
  redirect(308, '/cli#mcp');
};
