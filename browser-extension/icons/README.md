# Icon Files

This directory should contain the following icon files:

- `bee-16.png` - 16x16 pixel bee icon
- `bee-32.png` - 32x32 pixel bee icon  
- `bee-48.png` - 48x48 pixel bee icon
- `bee-128.png` - 128x128 pixel bee icon

## Creating Icons

You can create these icons using any image editor. The icons should be:

1. **Bee-themed**: Yellow and black colors representing a bumble bee
2. **Clear at small sizes**: Simple design that's recognizable at 16x16 pixels
3. **PNG format**: With transparency for better integration

## Placeholder Icons

For now, you can use emoji or create simple bee-colored squares:

- Background: Yellow (#FFC107)
- Accent: Black (#000000)
- Simple bee silhouette or just the 🐝 emoji

## Example using ImageMagick (if available)

```bash
# Create simple yellow squares with bee emoji (requires font support)
convert -size 16x16 xc:'#FFC107' -pointsize 12 -fill black -gravity center -annotate +0+0 '🐝' bee-16.png
convert -size 32x32 xc:'#FFC107' -pointsize 24 -fill black -gravity center -annotate +0+0 '🐝' bee-32.png
convert -size 48x48 xc:'#FFC107' -pointsize 36 -fill black -gravity center -annotate +0+0 '🐝' bee-48.png
convert -size 128x128 xc:'#FFC107' -pointsize 96 -fill black -gravity center -annotate +0+0 '🐝' bee-128.png
```