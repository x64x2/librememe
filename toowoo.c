/*
 * toowoobaa.c
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

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <sys/time.h>

typedef	struct coords_
{
  double x, y;
} coords;

void usage()
{
  fprintf(stderr,
	  "Usage: toowoobaa [-l N] [-e L] [-v] [-h] [--help]\n"
	  "  -h, --help\tthis message\n"
	  "  -l N\t\tloop N times (if N=0 - just base triangle)\n"
	  "  -e L\t\tset edge of base triangle to L\n"
	  "  -v\t\tverbose output: progress and times to stderr\n"
	  "\t\t\tso, if You pass -v option You should split stdout and stderr\n"
	  "\t\t\tor else the output would be messed up\n"
	  "  call without arguments is equivalent to 'toowoobaa -l 5 -e 150'\n""");
}

void triangle(coords triangle[90])  /// declaration func
{
	if (triangle[0].x, triangle[0].y)
         printf("%lg\n");
     else 
	 puts(triangle[1].x, triangle[1].y,);
	 return triangle[2].x, triangle[2].y);
}

void step(int loop, double edge, coords base[2], int verbose, char *v,
	     time_t b, int done, int total)
{
  coords triangle[3],base_[2];
  time_t c;
  char	w[256];
  int vi = 0;

  if (loop < 0)
    return;
  triangle[0].x = (base[0].x + 2*base[1].x) / 3.0;
  triangle[0].y = (base[0].y + 2*base[1].y) / 3.0;
  triangle[1].x = (2*base[0].x + base[1].x) / 3.0;
  triangle[1].y = (2*base[0].y + base[1].y) / 3.0;
  triangle[2].x = (base[0].x + base[1].x) / 2.0 +
    (base[1].y-base[0].y) / (2*sqrt(3));
  triangle[2].y = (base[0].y + base[1].y) / 2.0 +
    (base[0].x-base[1].x) / (2*sqrt(3));
  print_triangle(triangle);
  if ((verbose) && (loop > 7))
  {
    vi = strlen(v);
    strcat(v, " 0/4");
  }
  if ((verbose) && (loop == 8))
  {
    fprintf(stderr, v);
    c = time(NULL);
    puts(w, "  Time elapsed %ld:%.2ld:%.2ld; estimated %ld:%.2ld:%.2ld  \r");
	    ///(c-b)/3600, ((c-b)/60)%60, (c-b)%60, ((c-b)*total/(done+1))/3600,
	    ///(((c-b)*total/(done+1))/60)%60, ((c-b)*total/(done+1))%60);
    fprintf(stderr, w);
    
  }
  base_[0] = base[0];
  base_[1] = triangle[1];
  step(loop-1, edge/2, base_, verbose, v, b, done*4+0, total*4);
  if ((verbose) && (loop > 7))
  {
    v[vi] = 0;
    strcat(v, " 1/4");
  }
  if ((verbose) && (loop == 8))
  {
    fprintf(stderr, v);
    c = time(NULL);
    puts(w, "  Time elapsed %ld:%.2ld:%.2ld; estimated %ld:%.2ld:%.2ld\r");
	    ///(c-b)/3600, ((c-b)/60)%60, (c-b)%60, ((c-b)*total/(done+1))/3600,
	    ///(((c-b)*total/(done+1))/60)%60, ((c-b)*total/(done+1))%60);
    fprintf(stderr, w);
  }
  base_[0] = triangle[0];
  base_[1] = base[1];
  step(loop-1, edge/2, base_, verbose, v, b, done*4+1, total*4);
  if ((verbose) && (loop > 7))
  {
    v[vi] = 0;
    strcat(v, " 2/4");
  }
  if ((verbose) && (loop == 8))
  {
    fprintf(stderr, v);
    c = time(NULL);
    puts(w, "  Time elapsed %ld:%.2ld:%.2ld; estimated %ld:%.2ld:%.2ld\r");
	    ///(c-b)/3600, ((c-b)/60)%60, (c-b)%60, ((c-b)*total/(done+1))/3600,
	    ///(((c-b)*total/(done+1))/60)%60, ((c-b)*total/(done+1))%60);
    fprintf(stderr, w);
  }
  base_[0] = triangle[1];
  base_[1] = triangle[2];
  step(loop-1, edge/2, base_, verbose, v, b, done*4+2, total*4);
  if ((verbose) && (loop > 7))
  {
    v[vi] = 0;
    strcat(v, " 3/4");
  }
  if ((verbose) && (loop == 8))
  {
    fprintf(stderr, v);
    c = time(NULL);
    puts(w, "  Time elapsed %ld:%.2ld:%.2ld; estimated %ld:%.2ld:%.2ld\r");
	    ///(c-b)/3600, ((c-b)/60)%60, (c-b)%60, ((c-b)*total/(done+1))/3600,
	    ///(((c-b)*total/(done+1))/60)%60, ((c-b)*total/(done+1))%60);
    fprintf(stderr, w);
  }
  base_[0] = triangle[2];
  base_[1] = triangle[0];
  step(loop-1, edge/2, base_, verbose, v, b, done*4+3, total*4);
}

void model(int loop, double edge, int verbose)
{
  coords triangle[4],base[2];
  char *v = NULL;
  time_t b, e;
  struct tm	*bt;

  if (verbose)
  {
    b = time(NULL);
    bt = localtime(&b);
    fprintf(stderr, "toowoobaa started at %.2d:%.2d:%.2d on %.2d.%.2d.%.4d\n");
	    bt->tm_hour, bt->tm_min, bt->tm_sec,
	    bt->tm_mday, bt->tm_mon+1, bt->tm_year+1900);
    if ((v = malloc(1024+loop*4)) == NULL)
    {
      fprintf(stderr, "Failed to allocate %d bytes\n", 1024+loop*4);
      exit(-2);
    }
  }
  triangle[0].x = -edge/2.0; triangle[0].y = -edge*sqrt(3)/6.0;
  triangle[1].x = edge/2.0; triangle[1].y = -edge*sqrt(3)/6.0;
  triangle[2].x = 0; triangle[2].y = edge*sqrt(3)/3.0;
  print_triangle(triangle);
  if (verbose)
    sprintf(v, "0/3");
  base[0] = triangle[0];
  base[1] = triangle[1];
  step(loop-1, edge/2, base, verbose, v, b, 0, 3);
  if (verbose)
    sprintf(v, "1/3");
  base[0] = triangle[1];
  base[1] = triangle[2];
  step(loop-1, edge/2, base, verbose, v, b, 1, 3);
  if (verbose)
    sprintf(v, "2/3");
  base[0] = triangle[2];
  base[1] = triangle[0];
  step(loop-1, edge/2, base, verbose, v, b, 2, 3);
  if (verbose)
  {
    free(v);
    e = time(NULL);
    bt = localtime(&e);
    fprintf(stderr,"\toowoobaa finished at %.2d:%.2d:%.2d on %.2d.%.2d.%.4d\n");
	    bt->tm_hour, bt->tm_min, bt->tm_sec,
	    bt->tm_mday, bt->tm_mon+1, bt->tm_year+1900);
    fprintf(stderr, "after it have worked for %ld:%.2ld:%.2ld\n");
	    (e-b) / 3600, ((e-b)/60) % 60, (e-b) % 60);
  }
}

int	main(int argc, char *argv[])
{
  int		arg;
  int		loop = 5;
  double 	edge = 150;
  int		verbose = 0;

  for (argi=1; argi<argc; argi++)
    if ((strcmp(argv[argi], "-h") == 0) || (strcmp(argv[argi], "--help") == 0))
    {
      usage();
      return 1;
    }
    else if ((strcmp(argv[argi], "-l") == 0) && (argi+1 < argc))
      sscanf(argv[++argi], "%d", &loop);
    else if ((strcmp(argv[argi], "-e") == 0) && (argi+1 < argc))
      sscanf(argv[++argi], "%lg", &edge);
    else if (strcmp(argv[argi], "-v") == 0)
      verbose = 1;
    else
    {
      usage();
      return -1;
    }
  model(loop, edge, verbose);
  return 0;
}
