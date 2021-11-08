#include <Magick++.h>

#include <iostream>
#include <unistd.h>
#include <list>

int main(int argc, char const *argv[]) {
  std::string font = "futura";
  std::string caption = (std::string)argv[1];

  //std::cout << "loading" << std::endl;

  Magick::Blob blob;

  std::list<Magick::Image> frames;
  std::list<Magick::Image> coalesced;
  std::list<Magick::Image> captioned;
  try {
    //std::istreambuf_iterator<char> begin(std::cin), end;
    //std::string s(begin, end);
    //Magick::Blob test = Magick::Blob(s.c_str(), s.size());
    Magick::readImages(&frames, "-");
  } catch (Magick::WarningCoder &warning) {
    std::cerr << "Coder Warning: " << warning.what() << std::endl;
  } catch (Magick::Warning &warning) {
    std::cerr << "Warning: " << warning.what() << std::endl;
  }

  //std::cout << "coalescing" << std::endl;

  coalesceImages(&coalesced, frames.begin(), frames.end());

  //std::cout << "creating caption image" << std::endl;

  size_t width = coalesced.front().columns();

  //std::cout << std::to_string(width) << std::endl;

  std::string query(std::to_string(width - ((width / 25) * 2)) + "x");
  Magick::Image caption_image(Magick::Geometry(query), Magick::Color("white"));
  caption_image.fillColor("black");
  caption_image.alpha(true);
  caption_image.fontPointsize(width / 13);
  caption_image.textGravity(Magick::CenterGravity);
  caption_image.read("pango:<span font_family=\"" +
                       (font == "roboto" ? "Roboto Condensed" : font) +
                       "\" weight=\"" + (font != "impact" ? "bold" : "normal") +
                       "\">" + caption + "</span>");
  caption_image.extent(Magick::Geometry(width, caption_image.rows() + (width / 13)),
                       Magick::CenterGravity);

  //caption_image.write("./caption.png");

  //std::cout << "appending/processing" << std::endl;

  int i = 0;

  for (Magick::Image &image : coalesced) {
    //std::cout << "processing frame " + std::to_string(i) << std::endl;
    Magick::Image appended;
    std::list<Magick::Image> images;
    image.backgroundColor("white");
    images.push_back(caption_image);
    images.push_back(image);
    Magick::appendImages(&appended, images.begin(), images.end(), true);
    appended.repage();
    appended.magick("GIF");
    appended.animationDelay(image.animationDelay());
    appended.gifDisposeMethod(Magick::BackgroundDispose);
    appended.quantizeDither(false);
    appended.quantize();
    captioned.push_back(appended);
    i++;
  }

  //std::cout << "optimizing" << std::endl;
  //Magick::optimizeTransparency(captioned.begin(), captioned.end());

  //std::cout << "writing" << std::endl;
  Magick::writeImages(captioned.begin(), captioned.end(), "gif:-");

  //std::cout << "done" << std::endl;
  return 0;
}
