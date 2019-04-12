#include <stdio.h>
#include<stdlib.h>
int main()
{
    int arr[100];
    
   fork(); /* A */
   ( fork()  /* B */ &&  fork()  /* C */ ) ||   fork(); /* D */
   fork(); /* E */

  // printf("forked %d son from %d parent\n", getpid(), getppid());
   printf("%d\n", getpid());
   while(1);
   return 0;
}
