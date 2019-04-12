#include<stdio.h>
#include<stdlib.h>
int main() 
{ int arr[100]; 
    for(int i=0;i<5;i++) 
    { 
        int k = fork();
        if (k==0)
        { arr[i] = getpid();
           // printf("%d\n",arr[i]);
          //  printf("[son] pid %d from [parent] pid %d\n",arr[i],getppid()); 
           // exit(0); 
        }else
        {
            int *temp;
            temp = malloc(4*32768);
            printf("pid %d\n",getpid()); 
            for(int j=0;j<32760;j++)
            {temp[j]=getpid();
            }
            for(int j=0;j<32760;j++)
            {//printf("%d",temp[j]);
            }
        }
    }
     while(1)
          {
     }
     
} 
