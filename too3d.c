/*
 * too3d.c
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
#include <sys/time.h>

typedef	struct coords_
{
  double x, y, z;
} coords;

void	usage()
{
  fprintf(stderr, "Usage: kox3d [-l N] [-e L] [-h] [-v] [--help]\n"
	  "  -h, --help\tthis message\n"
	  "  -l N\t\tloop N times (if N=0 - just base tetrahedron)\n"
	  "  -e L\t\tset edge of base tetrahedron to L\n"
	  "  -v\t\tverbose output: progress and times to stderr\n"
	  "  call without arguments is equivalent to 'kox3d -l 5 -e 150'\n");
}

void print_all_triangles(coords tetrahedron[4], int loop)
{
  printf("%d %lg\n",loop,
	 tetrahedron[0].x, tetrahedron[0].y, tetrahedron[0].z,
	 tetrahedron[1].x, tetrahedron[1].y, tetrahedron[1].z,
	 tetrahedron[2].x, tetrahedron[2].y, tetrahedron[2].z);
  printf("%d %lg\n", loop,
	 tetrahedron[0].x, tetrahedron[0].y, tetrahedron[0].z,
	 tetrahedron[1].x, tetrahedron[1].y, tetrahedron[1].z,
	 tetrahedron[3].x, tetrahedron[3].y, tetrahedron[3].z);
  printf("%d %lg\n", loop,
	 tetrahedron[1].x, tetrahedron[1].y, tetrahedron[1].z,
	 tetrahedron[2].x, tetrahedron[2].y, tetrahedron[2].z,
	 tetrahedron[3].x, tetrahedron[3].y, tetrahedron[3].z);
  printf("%d %lg\n",loop,
	 tetrahedron[2].x, tetrahedron[2].y, tetrahedron[2].z,
	 tetrahedron[0].x, tetrahedron[0].y, tetrahedron[0].z,
	 tetrahedron[3].x, tetrahedron[3].y, tetrahedron[3].z);
}

void print_triangles(coords tetrahedron[4],int loop)
{
  printf("%d %lg\n",loop,
	 tetrahedron[0].x, tetrahedron[0].y, tetrahedron[0].z,
	 tetrahedron[1].x, tetrahedron[1].y, tetrahedron[1].z,
	 tetrahedron[3].x, tetrahedron[3].y, tetrahedron[3].z);
  printf("%d %lg\n",loop,
	 tetrahedron[1].x, tetrahedron[1].y, tetrahedron[1].z,
	 tetrahedron[2].x, tetrahedron[2].y, tetrahedron[2].z,
	 tetrahedron[3].x, tetrahedron[3].y, tetrahedron[3].z);
  printf("%d %lg\n",loop,
	 tetrahedron[2].x, tetrahedron[2].y, tetrahedron[2].z,
	 tetrahedron[0].x, tetrahedron[0].y, tetrahedron[0].z,
	 tetrahedron[3].x, tetrahedron[3].y, tetrahedron[3].z);
}

void step(int loop, double edge, coords base[3], int verbose, char *v,time_t b, int done, int total)
{
  int i;
  coords tetrahedron[4], n;
  double r;
  time_t c;
  char w[256];
  int vi = 0;

  if (loop < 0)
    return;
  if ((verbose) && (loop == 3))
  {
    fprintf(stderr, v);
    c = time(NULL);
    sprintf(w, "  Time elapsed %ld:%.2ld:%.2ld; estimated %ld:%.2ld:%.2ld  \r",
	    (c-b)/3600, ((c-b)/60)%60, (c-b)%60, ((c-b)*total/(done+1))/3600,
	    (((c-b)*total/(done+1))/60)%60, ((c-b)*total/(done+1))%60);
    fprintf(stderr, w);
  }
  for (i=0; i<3; i++)
  {
    tetrahedron[i].x = (base[i].x + base[(i+1) % 3].x) / 2.0;
    tetrahedron[i].y = (base[i].y + base[(i+1) % 3].y) / 2.0;
    tetrahedron[i].z = (base[i].z + base[(i+1) % 3].z) / 2.0;
  }
  n.x = ((tetrahedron[1].y-tetrahedron[0].y)*
	 (tetrahedron[2].z-tetrahedron[0].z) -
	 (tetrahedron[1].z-tetrahedron[0].z)*
	 (tetrahedron[2].y-tetrahedron[0].y));
  n.y = ((tetrahedron[1].z-tetrahedron[0].z)*
	 (tetrahedron[2].x-tetrahedron[0].x) -
	 (tetrahedron[1].x-tetrahedron[0].x)*
	 (tetrahedron[2].z-tetrahedron[0].z));
  n.z = ((tetrahedron[1].x-tetrahedron[0].x)*
	 (tetrahedron[2].y-tetrahedron[0].y) -
	 (tetrahedron[1].y-tetrahedron[0].y)*
	 (tetrahedron[2].x-tetrahedron[0].x));
  r = sqrt(n.x*n.x+n.y*n.y+n.z*n.z);
  n.x /= r; n.y /= r; n.z /= r;
  tetrahedron[3].x = (base[0].x + base[1].x + base[2].x) / 3.0 +
    n.x * sqrt(2.0/3.0) * edge;
  tetrahedron[3].y = (base[0].y + base[1].y + base[2].y) / 3.0 +
    n.y * sqrt(2.0/3.0) * edge;
  tetrahedron[3].z = (base[0].z + base[1].z + base[2].z) / 3.0 +
    n.z * sqrt(2.0/3.0) * edge;
  print_triangles(tetrahedron, loop);
  if ((verbose) && (loop > 3))
  {
    vi = strlen(v);
    strcat(v, " 0/3");
  }
  base[0] = tetrahedron[0];
  base[1] = tetrahedron[1];
  base[2] = tetrahedron[3];
  step(loop-1, edge/2.0, base, verbose, v, b, done*3, total*3);
  if ((verbose) && (loop > 3))
  {
    v[vi] = 0;
    strcat(v, " 1/3");
  }
  base[0] = tetrahedron[1];
  base[1] = tetrahedron[2];
  base[2] = tetrahedron[3];
  step(loop-1, edge/2.0, base, verbose, v, b, done*3+1, total*3);
  if ((verbose) && (loop > 3))
  {
    v[vi] = 0;
    strcat(v, " 2/3");
  }
  base[0] = tetrahedron[2];
  base[1] = tetrahedron[0];
  base[2] = tetrahedron[3];
  step(loop-1, edge/2.0, base, verbose, v, b, done*3+2, total*3);
}

void model(int loop, double edge, int verbose)
{
  coords tetrahedron[4], base[3];
  char *v = NULL;
  time_t b, e;
  struct tm	*bt;

  if (verbose)
  {
    b = time(NULL);
    bt = localtime(&b);
    fprintf(stderr, "toowoobaa started at %.2d:%.2d:%.2d on %.2d.%.2d.%.4d\n",
	    bt->tm_hour, bt->tm_min, bt->tm_sec,
	    bt->tm_mday, bt->tm_mon+1, bt->tm_year+1900);
    if ((v = malloc(1024+loop*4)) == NULL)
    {
      fprintf(stderr, "Failed to allocate %d bytes\n", 1024+loop*4);
      exit(-2);
    }
  }
  tetrahedron[0].x = -edge/2.0; tetrahedron[0].y = -edge/(2*sqrt(3));
  tetrahedron[0].z = -sqrt(2.0/3.0)*edge/4.0;
  tetrahedron[1].x = edge/2.0; tetrahedron[1].y = -edge/(2*sqrt(3));
  tetrahedron[1].z = -sqrt(2.0/3.0)*edge/4.0;
  tetrahedron[2].x = 0; tetrahedron[2].y = edge*sqrt(3)/2.0-edge/(2*sqrt(3));
  tetrahedron[2].z = -sqrt(2.0/3.0)*edge/4.0;
  tetrahedron[3].x = 0; tetrahedron[3].y = 0;
  tetrahedron[3].z = sqrt(2.0/3.0)*edge*3/4.0;
  print_all_triangles(tetrahedron, loop);
  if (verbose)
    fprintf(v, "0/4");
  base[0] = tetrahedron[0];
  base[1] = tetrahedron[2];
  base[2] = tetrahedron[1];
  step(loop-1, edge/2.0, base, verbose, v, b, 0, 4);
  if (verbose)
    fprintf(v, "1/4");
  base[0] = tetrahedron[0];
  base[1] = tetrahedron[1];
  base[2] = tetrahedron[3];
  step(loop-1, edge/2.0, base, verbose, v, b, 1, 4);
  if (verbose)
    fprintf(v, "2/4");
  base[0] = tetrahedron[1];
  base[1] = tetrahedron[2];
  base[2] = tetrahedron[3];
  step(loop-1, edge/2.0, base, verbose, v, b, 2, 4);
  if (verbose)
    fprintf(v, "3/4");
  base[0] = tetrahedron[2];
  base[1] = tetrahedron[0];
  base[2] = tetrahedron[3];
  step(loop-1, edge/2.0, base, verbose, v, b, 3, 4);
  if (verbose)
  {
    free(v);
    e = time(NULL);
    bt = localtime(&e);
    fprintf(stderr, "\too3d finished at %.2d:%.2d:%.2d on %.2d.%.2d.%.4d\n",
	    bt->tm_hour, bt->tm_min, bt->tm_sec,
	    bt->tm_mday, bt->tm_mon+1, bt->tm_year+1900);
    fprintf(stderr, "after it have worked for %ld:%.2ld:%.2ld\n",
	    (e-b) / 3600, ((e-b)/60) % 60, (e-b) % 60);
  }
}

int	main(int argc, char *argv[])
{
  int arg;
  int loop = 5;
  double edge = 150;
  int verbose = 0;
  for (arg=1; arg<argc; arg++)
    if ((strcmp(argv[argi], "-h") == 0) || (strcmp(argv[arg], "--help") == 0))
    {
      usage();
      return 1;
    }
    else if ((strcmp(argv[arg], "-l") == 0) && (arg+1 < argc))
      sscanf(argv[++arg], "%d", &loop);
    else if ((strcmp(argv[arg], "-e") == 0) && (arg+1 < argc))
      sscanf(argv[++arg], "%lg", &edge);
    else if (strcmp(argv[arg], "-v") == 0)
      verbose = 1;
    else
    {
      usage();
      return -1;
    }
  model(loop, edge, verbose);
  return 0;
}
