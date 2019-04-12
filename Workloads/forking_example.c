#include<stdio.h>
#include<stdlib.h>
int main() 
{ int arr[100]; 
    for(int i=0;i<5;i++) 
    { 
        if(fork() == 0) 
        { arr[i] = getpid();
          //  printf("%d\n",arr[i]);
            printf("[son] pid %d from [parent] pid %d\n",arr[i],getppid()); 
           // exit(0); 
        }else
        {
            int temp[32760];
           // printf("pid %d\n",getpid()); 
            for(int j=0;j<32760;j++)
                temp[j]=0;
        }
    }
     while(1)
          {
     }
     
} 
