# tfviz website

A static, multi-page site with no JavaScript, cookies or analytics. Pages are rendered by `scripts/site` (Go standard library only) and the output in `public/` is committed. Hosts serve it as is, with no build step.

```text
site/
  site.json          base URL, version and repository link
  templates/         layout, icons and one template per page
  static/            stylesheet, font, favicon, images and demo reports
  public/            generated: the deployable site (do not edit)
  vercel.json        generated: Vercel settings and headers
  wrangler.jsonc     Cloudflare Workers static assets settings
```

## Editing

1. Change a template, the stylesheet or `site.json`.
2. Run `make site`.
3. Commit `site/` with your change. CI runs `make site-check` and fails if `public/` is stale.

The generator tests check that:
- every local link resolves
- pages contain no inline scripts or styles
- every page gets the same security headers on both hosts

### Screenshots and demo reports

```bash
make site-assets ICONS=~/Downloads/aws-icons
```

This needs the unzipped AWS Architecture Icons pack. It renders the demo reports with the real CLI, using tfviz's own symbols so no AWS icon files are published. It then captures the screenshots, which are diagrams that use the official icons, as WebP and builds `og.png`. Run it whenever the report interface changes visibly.

## Security headers

`scripts/site/headers.go` is the single source for both hosts. It generates `public/_headers` for Cloudflare and `vercel.json` for Vercel.

- **Pages.** `default-src 'none'`, with styles, images and the font from the site only, and no framing. The same policy is also set in each page's `<meta>` tag, so it holds even if a host drops the header.
- **Demo reports (`/reports/`).** These keep their own strict, hash-based policy. The header adds only `frame-ancestors 'none'`.
- **Every response.** `nosniff`, `no-referrer`, a restrictive `Permissions-Policy`, `Cross-Origin-Opener-Policy` and HSTS.

## Deploying

### Vercel

1. Import the GitHub repository as a new project.
2. Set **Root Directory** to `site`. This can't be set in `vercel.json`.
3. Leave the framework as **Other** and the build command empty. `vercel.json` sets the output directory to `public`, skips installation and applies the headers.

Or use the CLI, after `vercel login`: `cd site && vercel deploy --prod`.

### Cloudflare Workers

From `site/`, after `npx wrangler login`:

```bash
npx wrangler deploy
```

This deploys an assets-only Worker: no Worker code runs. Cloudflare applies `public/_headers`, serves `404.html` for unknown paths and redirects `/features` to `/features/`.

## Choosing the domain

When the domain is ready:

1. Set `base_url` in `site.json`, for example `"https://tfviz.dev"`.
2. Run `make site` and commit.

This adds canonical links, Open Graph URLs and image, and `sitemap.xml`. Until then they are left out rather than pointing at a guess.

Check the headers after the first deploy:

```bash
curl -sI https://<domain>/ | grep -i -E "content-security|strict-transport|x-frame"
```

## Licences

Inter is self-hosted under the SIL Open Font License 1.1. The licence is published at `/fonts/OFL.txt`.
