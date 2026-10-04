# Official service icons

By default, reports use tfviz's own small symbols, one per service family. If you want the official AWS service icons instead, download the icon pack yourself and point tfviz at it:

```bash
tfviz plan --input plan.json --output report.html --icons ~/Downloads/aws-icons
```

`--icons` works with `plan`, `state` and `explore`.

## Getting the AWS pack

1. Download the asset package from the [AWS Architecture Icons](https://aws.amazon.com/architecture/icons/) page.
2. Unzip it.
3. Pass the unzipped folder to `--icons`. tfviz searches its subfolders, so you don't need to move or rename anything.

The resource-type mapping was checked against the 31 July 2026 release. It uses resource icons where AWS publishes one (for example NAT gateway or S3 bucket) and service icons otherwise. AWS has no icon for security groups or security group rules, so those keep tfviz's symbol. Any type without a match does the same.

## Why tfviz doesn't include the icons

AWS allows customers and partners to use its icons to create architecture diagrams. That is what you do when you generate a report. AWS doesn't grant a licence to redistribute the icons inside software, so tfviz never ships them. Instead:

- you download the pack under AWS's terms
- tfviz copies the icons it needs into each report you generate

Check the pack's terms before you share reports outside your organisation.

## What goes into the report

- **Only what's used.** Only icons for resource types present in the report are embedded, each once.
- **Offline.** Icons are stored as `data:` images, so reports stay a single offline file under the same content security policy.
- **Images only.** They are displayed only as images. Browsers don't run scripts or fetch anything from an SVG shown that way.
- **Limits.** tfviz reads only regular `.svg` files. It skips hidden files, macOS `__MACOSX` folders and symbolic links. It refuses files over 64 KiB, folders with more than 50,000 entries, and a set of icons over 2 MiB.
- **Wrong folder.** If none of the expected file names is found, tfviz stops with an error rather than silently producing a report without icons.
- **Safe-share.** `--safe-share` keeps icons, because they identify only the service type, which safe-share keeps anyway.

Azure and Google Cloud packs will work the same way once those providers are supported.
