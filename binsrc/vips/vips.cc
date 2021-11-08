#include <iostream>
#include <vips/vips8>

using namespace vips;

int main(int argc, char **argv) {
  if (vips_init(argv[0])) vips_error_exit(NULL);

  /*VImage in = VImage::new_from_file(
      "./marioheadtest.gif",
      VImage::option()->set("access", VIPS_ACCESS_SEQUENTIAL)->set("n", -1));*/
  VImage in = VImage::new_from_source(VSource::new_from_descriptor(0),
                                      "[n=-1,access=sequential]");

  int width = in.width();

  int size = width / 10;
  char font_string[12 + sizeof(size)];

  std::sprintf(font_string, "futura bold %d", size);

  int textWidth = width - ((width / 25) * 2);

  std::string captionText =
      "<span background=\"white\">" + (std::string)argv[1] + "</span>";

  VImage text =
      VImage::text(captionText.c_str(), VImage::option()
                                            ->set("rgba", TRUE)
                                            ->set("align", VIPS_ALIGN_CENTRE)
                                            ->set("font", font_string)
                                            ->set("width", textWidth));

  std::vector<double> fullAlpha = {0, 0, 0, 0};

  VImage caption =
      text.relational_const(VIPS_OPERATION_RELATIONAL_EQUAL, fullAlpha)
          .bandand()
          .ifthenelse({255, 255, 255, 255}, text)
          .gravity(VIPS_COMPASS_DIRECTION_CENTRE, width, text.height() + size,
                   VImage::option()->set("extend", VIPS_EXTEND_WHITE));

  int pageHeight = in.get_int(VIPS_META_PAGE_HEIGHT);
  std::vector<VImage> gif;

  for (int i; i < (in.height() / pageHeight); i++) {
    VImage cropped = in.crop(0, i * pageHeight, width, pageHeight)
                         .colourspace(VIPS_INTERPRETATION_sRGB);
    gif.push_back(caption.join(
        !cropped.has_alpha() ? cropped.bandjoin(255) : cropped,
        VIPS_DIRECTION_VERTICAL,
        VImage::option()->set("background", 0xffffff)->set("expand", TRUE)));
  }

  VImage result = VImage::arrayjoin(gif, VImage::option()->set("across", 1));
  result.set(VIPS_META_PAGE_HEIGHT, pageHeight + caption.height());

  // result.write_to_file("./vipsout.gif", VImage::option()->set("dither", 0));

  result.write_to_target(".gif", VTarget::new_to_descriptor(1),
                         VImage::option()->set("dither", 0));

  vips_shutdown();

  return 0;
}
