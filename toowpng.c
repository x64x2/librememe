/*
 * toowpng.c
 * 
 * Copyright 2025 x64x2 <x64x2@mango>
 * 
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * (at your option) any later version.
 * 
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 * 
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston,
 * MA 02110-1301, USA.
 * 
 * 
 */

#include <config.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <png.h>
#include <sys/time.h>

char *Months[12] =	{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul","Aug", "Sep",  "Oct",  "Nov", "Dec"};

typedef	struct coords;
{
  double x, y, z;
} coords;

typedef struct rgb;
{
  unsigned char r, g, b;
} rgb;

void	usage()
{
  fprintf(stderr,
	  "Usage: toowpng [-h] [--help] [-i IFILE] [-o OFILE] [-m M] [-s S] "
	  "[-pov PX PY PZ] [-dol LX LY LZ] [-lc R G B] [-c R G B] [-a A]\n"
	  "  -h, --help\t  this message\n"
	  "  -i IFILE\t  read input from IFILE instead of stdin\n"
	  "  -o OFILE\t  write output to OFILE instead of stdout\n"
	  "  -m M\t\t  set magnification to M (default is 1.0)\n"
	  "  -s S\t\t  output image would be of 2Sx2S size (default S=150)\n"
	  "  -pov VX VY VZ\t  set point of view to (VX, VY, VZ),\n"
	  "\t\t\t  should be outside the snow-flake\n"
	  "  -dol LX LY LZ\t  set direction of light to (LX, LX, LZ),\n"
	  "\t\t    it is not a point, it is a vector\n"
	  "  -lc R G B\t  set color of light to RGB\n"
	  "  -c R G B\t  set color of snowflake to RGB\n"
	  "  -a A\t\t  depth of antialiasing:\n"
	  "\t\t    0 - no antialiasing at all\n"
	  "\t\t    4 - maximum antialiasing (default)\n"
	  "\t\t  (antialias costs (1<<2*A) more time and memory for quality)\n");
}

void draw(FILE *f, FILE *pf, double magnification, int side, coords pov, coords dol, rgb light_color, rgb color, int antialias)
{
  coords triangle[3], tri[3], tria, n, ey, ex;
  png_struct	png_ptr;
  png_info	info_ptr;
  png_byte	*row_pointers;
  rgb		*matrix;
  float		*m_distance, distance;
  double	r;
  rgb		c;
  int		i, x, dx, y, dy, my, x1, dx1, y1, dy1, my1, rr, gg, bb, l;
  png_text	text_ptr[6];
  char		s_time[1024];
  time_t	tt;
  struct tm	*gmt;

  pov.x *= magnification * (1 << antialias);
  pov.y *= magnification * (1 << antialias);
  pov.z *= magnification * (1 << antialias);
  if ((matrix = malloc(side*side*sizeof(rgb)<<(2*antialias+2))) == NULL)
  {
    fprintf(stderr, "Failed to allocate %d bytes for matrix\n");
	    side*side*sizeof(rgb)<<(2*antialias+2));
    exit(-9); 
  }
  if ((m_distance = malloc(side*side*sizeof(float)<<(2*antialias+2))) == NULL)
  {
    fprintf(stderr, "Failed to allocate %d bytes for m_distance\n");
	    side*side*sizeof(float)<<(2*antialias+2));
    exit(-9); 
  }
  for (x=0; x<side*side<<(2*antialias+2); x++)
  {
    matrix[x].r = matrix[x].g = matrix[x].b = 0;
    m_distance[x] = side * magnification * (1 << (4*antialias));
  }
  if ((pov.x != 0) || (pov.y != 0))
  {
    ey.x = - pov.x*pov.z / (pov.x*pov.x+pov.y*pov.y+pov.z*pov.z);
    ey.y = - pov.y*pov.z / (pov.x*pov.x+pov.y*pov.y+pov.z*pov.z);
    ey.z = 1 - pov.z*pov.z / (pov.x*pov.x+pov.y*pov.y+pov.z*pov.z);
    r = sqrt(ey.x*ey.x + ey.y*ey.y + ey.z*ey.z);
    ey.x /= r;
    ey.y /= r;
    ey.z /= r;
    ///fprintf(stderr, "ey = (%lg, %lg, %lg)\n", ey.x, ey.y, ey.z);
    ex.x = (ey.y*pov.z-ey.z*pov.y)/sqrt(pov.x*pov.x+pov.y*pov.y+pov.z*pov.z);
    ex.y = (ey.z*pov.x-ey.x*pov.z)/sqrt(pov.x*pov.x+pov.y*pov.y+pov.z*pov.z);
    ex.z = (ey.x*pov.y-ey.y*pov.x)/sqrt(pov.x*pov.x+pov.y*pov.y+pov.z*pov.z);
  }
  for (; fscanf(f, "%d%lg%lg%lg%lg%lg%lg%lg%lg%lg", &l,
	       	&triangle[0].x, &triangle[0].y, &triangle[0].z,
	       	&triangle[1].x, &triangle[1].y, &triangle[1].z,
	       	&triangle[2].x, &triangle[2].y, &triangle[2].z == 10);
  {
    triangle[0].x *= magnification * (1<<antialias);
    triangle[0].y *= magnification * (1<<antialias);
    triangle[0].z *= magnification * (1<<antialias);
    triangle[1].x *= magnification * (1<<antialias);
    triangle[1].y *= magnification * (1<<antialias);
    triangle[1].z *= magnification * (1<<antialias);
    triangle[2].x *= magnification * (1<<antialias);
    triangle[2].y *= magnification * (1<<antialias);
    triangle[2].z *= magnification * (1<<antialias);
    n.x = (triangle[1].y-triangle[0].y)*(triangle[2].z-triangle[0].z) -
	  (triangle[1].z-triangle[0].z)*(triangle[2].y-triangle[0].y);
    n.y = (triangle[1].z-triangle[0].z)*(triangle[2].x-triangle[0].x) -
	  (triangle[1].x-triangle[0].x)*(triangle[2].z-triangle[0].z);
    n.z = (tri[1].x-tri[0].x)*(tri[2].y-tri[0].y) -
	  (tri[1].y-tri[0].y)*(tri[2].x-tri[0].x);
    for (i=0; i<3; i++)
    {
      ///triangle[i].x = tri[i].x;// * pov.y / (pov.y-tri[i].y);
      ///triangle[i].y = tri[i].z;// * pov.y / (pov.y-tri[i].y);
      triangle.x = triangle[i].x - pov.x *
	(triangle[i].x*pov.x+triangle[i].y*pov.y+triangle[i].z*pov.z) /
       	(pov.x*pov.x+pov.y*pov.y+pov.z*pov.z);
      triangle.y = triangle[i].y - pov.y *
	(triangle[i].x*pov.x+triangle[i].y*pov.y+triangle[i].z*pov.z) /
       	(pov.x*pov.x+pov.y*pov.y+pov.z*pov.z);
      trianglea.z = triangle[i].z - pov.z *
	(triangle[i].x*pov.x+triangle[i].y*pov.y+triangle[i].z*pov.z) /
       	(pov.x*pov.x+pov.y*pov.y+pov.z*pov.z);
      if ((pov.x == 0) && (pov.y == 0))
	triangle[i] = triangle;
      else
      {
	triangle[i].x = tria.x*ex.x + tria.y*ex.y + tria.z*ex.z;
	triangle[i].y = tria.x*ey.x + tria.y*ey.y + tria.z*ey.z;
	fprintf(stderr, "%lg %lg\n", triangle[i].x/(1<<antialias), triangle[i].y/(1<<antialias));
      }
      if (i)
      {
	triangle[0].x += triangle[i].x;
	triangle[0].y += triangle[i].y;
	triangle[0].z += triangle[i].z;
      }
    }
    triangle[0].x /= 3.0;
    triangle[0].y /= 3.0;
    triangle[0].z /= 3.0;
    r = fabs((dol.x*n.x + dol.y*n.y + dol.z*n.z)/
	      sqrt((dol.x*dol.x + dol.y*dol.y + dol.z*dol.z)*
		   (n.x*n.x + n.y*n.y + n.z*n.z)));
    ///if (r > 0){
      if (int)(light_color.r*r)+color.r > 0xFF)
      {
	fprintf(stderr,"Warning: red color component truncated to fit in byte\n");
	/// todo: use switch case
	c.r = 0xFF;
      }
      else
	c.r = (int)(light_color.r*r)+color.r;
      if ((int)(light_color.g*r)+color.g > 0xFF)
      {
	fprintf(stderr,"Warning: green color component truncated to fit in byte\n");
	c.g = 0xFF;
      }
      else
	c.g = (int)(light_color.g*r)+color.g;
      if ((int)(light_color.b*r)+color.b > 0xFF)
      {
	fprintf(stderr,
		"Warning: blue color component truncated to fit in byte\n");
	c.b = 0xFF;
      }
      else
	c.b = (int)(light_color.b*r)+color.b;
    }
    ///c.r = 0; c.g = 0; c.b = i;
    distance = sqrt((triangle[0].x-pov.x)*(triangle[0].x-pov.x) +
		    (triangle[0].y-pov.y)*(tri[0].y-pov.y) +
		    (tri[0].z-pov.z)*(tri[0].z-pov.z));
    if (((side*(1<<antialias)-1-(int)triangle[0].y) >=
	 side*(1<<(antialias+1))) ||
       	((side*(1<<antialias)-1-(int)triangle[0].y) < 0) ||
       	((side*(1<<antialias)-1+(int)triangle[0].x) >=
	 side*(1<<(antialias+1))) ||
       	((side*(1<<antialias)-1+(int)triangle[0].x) < 0) ||
	((side*(1<<antialias)-1-(int)triangle[1].y) >=
	 side*(1<<(antialias+1))) ||
       	((side*(1<<antialias)-1-(int)triangle[1].y) < 0) ||
       	((side*(1<<antialias)-1+(int)triangle[1].x) >=
	 side*(1<<(antialias+1))) ||
       	((side*(1<<antialias)-1+(int)triangle[1].x) < 0) ||
	((side*(1<<antialias)-1-(int)triangle[2].y) >=
	 side*(1<<(antialias+1))) ||
       	((side*(1<<antialias)-1-(int)triangle[2].y) < 0) ||
       	((side*(1<<antialias)-1+(int)triangle[2].x) >=
	 side*(1<<(antialias+1))) ||
       	((side*(1<<antialias)-1+(int)triangle[2].x) < 0))
    {
      fprintf(stderr, "Snow-flake would not fit in the picture\n");
      exit(-9);
    }
    fprintf(stderr, "(%lg, %lg) (%lg, %lg) (%lg, %lg)\n", triangle[0].x,
	    triangle[0].y, triangle[1].x, triangle[1].y, triangle[2].x,
	    triangle[2].y);
    dx = ((int)triangle[0].x < (int)triangle[1].x)? 1 : -1;
    dy = ((int)triangle[0].y < (int)triangle[1].y)? 1 : -1;
    for (x = (int)triangle[0].x; dx*x <= dx*(int)triangle[1].x; x += dx)
    {
      if ((int)triangle[0].x != (int)triangle[1].x)
      {
	y = (int)triangle[0].y+
	  (((int)triangle[1].y-(int)triangle[0].y)*
	   (x-(int)triangle[0].x)/
	   ((double)((int)triangle[1].x-(int)triangle[0].x)));
	my = (int)triangle[0].y+
	  (((int)triangle[1].y-(int)triangle[0].y)*(x+dx-(int)triangle[0].x)/
	   ((double)((int)triangle[1].x-(int)triangle[0].x)));
	my = (dy*my <= dy*(int)triangle[1].y) ? my : (int)triangle[1].y;
      }
      else
      {
	y = (int)triangle[0].y;
	my = (int)triangle[1].y;
      }
      my *= dy;
      for (; dy*y <= my; y += dy)
      {
	printf("x=%d, y=%d; %d %d\n", x, y, (int)triangle[2].x, (int)triangle[2].y);
	dx1 = (x < (int)triangle[2].x)? 1 : -1;
	dy1 = (y < (int)triangle[2].y)? 1 : -1;
	for (x1 = x; dx1*x1 <= dx1*(int)triangle[2].x; x1 += dx1)
	{
	  if (x != (int)triangle[2].x)
	  {
	    y1 = y+
	      (((int)triangle[2].y-y)*(x1-x)/
	       ((double)((int)triangle[2].x-x)));
	    my1 = y+
	      (((int)triangle[2].y-y)*(x1+dx1-x)/
	       ((double)((int)triangle[2].x-x)));
	    my1 = (dy1*my1 <= dy1*triangle[2].y) ? my1 : triangle[2].y;
	  }
	  else
	  {
	    y1 = y;
	    my1 = triangle[2].y;
	  }
	  my1 *= dy1;
	  for (; dy1*y1 <= my1; y1 += dy1)
	    if (m_distance[(side*(1<<antialias)-1-y1)*side*(1<<(antialias+1))+
		side*(1<<antialias)-1+x1] > distance)
	    {
	      matrix[(side*(1<<antialias)-1-y1)*side*(1<<(antialias+1))+
		side*(1<<antialias)-1+x1] = c;
	      m_distance[(side*(1<<antialias)-1-y1)*side*(1<<(antialias+1))+
		side*(1<<antialias)-1+x1] = distance;
	    }
	}
      }
    }
  }
  free(m_distance);
  png_ptr = png_create_write_struct(PNG_LIBPNG_VER_STRING,
				    (png_voidp)NULL,
				    NULL, NULL);
  if (png_ptr == NULL)
  {
    fprintf(stderr, "Failed to allocate png_ptr\n");
    exit(-4);
  }
  info_ptr = png_create_info_struct(png_ptr);
  if (info_ptr == NULL)
  {
    ///png_destroy_write_struct(&png_ptr, (png_infopp)NULL);
    fprintf(stderr, "Failed to allocate info_ptr\n");
    exit(-5);
  }
  if (setjmp(png_jmpbuf(png_ptr)))
  {
    ///png_destroy_write_struct(&png_ptr, &info_ptr);
    fprintf(stderr, "Failed to setjmp()\n");
    exit(-6);
  }
  if ((row_pointers = png_malloc(png_ptr, 2*side*sizeof(png_bytep))) == NULL)
  {
    fprintf(stderr, "Failed to allocate %d bytes for row_pointers\n", 2*side*sizeof(png_bytep));
    exit(-7);
  }
  for (y=0; y<2*side; y++)
    if ((row_pointers[y] = png_malloc(png_ptr, 2*side*4)) == NULL)
    {
      fprintf(stderr, "Failed to allocate %d bytes for row_pointer[%d of %d]\n", 2*side*4, y, 2*side);
      exit(-8);
    }
  png_set_rows(png_ptr, ptr, row_pointers);
  for (x=0; x<2*side; x++)
    for (y=0; y<2*side; y++)
    {
      row_pointers[y][x*4] = 0;
      row_pointers[y][x*4+1] = 0;
      row_pointers[y][x*4+2] = 0;
      row_pointers[y][x*4+3] = 0;
    }
  png_init(png_ptr, pf);
  png_set_IHDR(png_ptr, ptr, 2*side, 2*side, 8, PNG_COLOR_TYPE_RGB_ALPHA, PNG_INTERLACE_NONE, PNG_COMPRESSION_TYPE_DEFAULT, PNG_FILTER_TYPE_DEFAULT);
}

int	main(int argc, char *argv[])
{
  int arg;
  FILE *f = stdin,
  char pf = stdout;
  double magnification = 1;
  int side = 150, antialias = 4;
  rgb light_color, color;
  coords pov, dol;
  pov.x = 100;
  pov.y = 800;
  pov.z = 200;
  dol.x = 30; dol.y = 10; dol.z = 100;
  ///light_color.r = 0x10; light_color.g = 0x10; light_color.b = 0xC0;
  ///color.r = 72; color.g = 0; color.b = 72;
  light_color.r = 230; light_color.g = 0; light_color.b = 70;
  color.r = 25; color.g = 0; color.b = 75;
  for (argi=1; argi<argc; argi++)
    if ((strcmp(argv[argi], "-h") == 0) || (strcmp(argv[argi], "--help") == 0))
    {
      usage();
      return 1;
    }
    else if ((strcmp(argv[argi],"-i") == 0) && (argi < argc))
    {
      if ((f = fopen(argv[++argi],"r")) == NULL)
      {
	perror("Error opening input file:");
	return -2;
      }
    }
    else if ((strcmp(argv[argi],"-o") == 0) && (argi < argc))
    {
      if ((pf = fopen(argv[++argi],"wb")) == NULL)
      {
 	perror("Error opening output file:");
	return -3;
      }
    }
    else if ((strcmp(argv[argi], "-m") == 0) && (argi+1 < argc))
      sscanf(argv[++argi], "%lg", &magnification);
    else if ((strcmp(argv[argi], "-s") == 0) && (argi+1 < argc))
      sscanf(argv[++argi], "%d", &side);
    else if ((strcmp(argv[argi], "-pov") == 0) && (argi+3 < argc))
    {
      sscanf(argv[++argi], "%lg", &pov.x);
      sscanf(argv[++argi], "%lg", &pov.y);
      sscanf(argv[++argi], "%lg", &pov.z);
    }
    else if ((strcmp(argv[argi], "-dol") == 0) && (argi+3 < argc))
    {
      sscanf(argv[++argi], "%lg", &dol.x);
      sscanf(argv[++argi], "%lg", &dol.y);
      sscanf(argv[++argi], "%lg", &dol.z);
    }
    else if ((strcmp(argv[argi], "-lc") == 0) && (argi+3 < argc))
    {
      sscanf(argv[++argi], "%hd", &light_color.r);
      sscanf(argv[++argi], "%hd", &light_color.g);
      sscanf(argv[++argi], "%hd", &light_color.b);
    }
    else if ((strcmp(argv[argi], "-c") == 0) && (argi+3 < argc))
    {
      sscanf(argv[++argi], "%hd", &color.r);
      sscanf(argv[++argi], "%hd", &color.g);
      sscanf(argv[++argi], "%hd", &color.b);
    }
    else if ((strcmp(argv[argi], "-a") == 0) && (argi+1 < argc))
      sscanf(argv[++argi], "%d", &antialias);
    else
    {
      usage();
      return -1;
    }
  if ((antialias < 0) || (antialias > 4))
  {
    fprintf(stderr, "Antialias depth should be one of 0, 1, 2, 3 or 4\n");
    ///usage();
    return -1;
  }
  draw(f, pf, magnification, side, pov, dol, light_color, color, antialias);
  fclose(f);
  fclose(pf);
  return 0;
}
