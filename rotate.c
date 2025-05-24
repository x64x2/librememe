#include <stdio.h>
#include <stdlib.h>
#include <math.h>

#ifndef PI
#define PI	3.1415926535897932384626433832795029L

void main()
{
  double	x, y, a;
  int		i;
  for (a=0, i=0; a<360; a+=10, i+=1)
  {
    x = cos(PI*a/180) * 800;
    y = sin(PI*a/180) * 800;
    printf("./too3d -e 133 | ./toowpng -s 100 -pov %lg %lg 200 -a 2 -o kox3_%d.png\n", x, y, i);
  }
}
