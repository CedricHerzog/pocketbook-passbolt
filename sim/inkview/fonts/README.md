The reader's default fonts (inkview.h), so that the preview measures and wraps
text as the reader does:

- `LiberationSans.ttf`, `LiberationSans-Bold.ttf`: Liberation 1.04, the
  firmware's own files from `/ebrmain/fonts` (GPLv2 with the font exception,
  `LICENSE-Liberation`).
- `LiberationMono.ttf`: Liberation Mono 2.1.5 (SIL OFL 1.1,
  `LICENSE-LiberationMono-OFL-1.1`). It is not the firmware's own copy, but
  Liberation Mono has the same fixed advance width in every version, so
  passwords wrap the same.
