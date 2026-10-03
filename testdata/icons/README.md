# Placeholder icons

These files stand in for an official provider icon pack in tests. They copy only the pack's folder layout and file names; the artwork is plain shapes drawn for tfviz, not provider icons. tfviz never ships provider icons. Users download the official pack and pass its folder with `--icons`.

`aws/` mirrors the AWS Architecture Icons layout:

- A 48 and a 64 version of one icon, so tests can check that the 48 is preferred.
- A `__MACOSX` entry that must be ignored.
