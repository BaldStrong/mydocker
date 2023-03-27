package nsenter

/*
#include <errno.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>

// 这里的attribute ((constructor ））指的是， 一旦这个包被引用，那么这个函数就会被自动执行
// 类似于构造函数，会在程序一启动的时候运行
__attribute__((constructor)) void enter_namespace(void) {
    char *mydocker_pid;
    mydocker_pid = getenv("mydocker_pid");
    if(mydocker_pid){
        //fprintf(stdout,"got mydocker_pid=%s\n",mydocker_pid);
    }else{
        //fprintf(stdout,"missing mydocker pid env skip nsenter\n");
        return;
    }

    char *mydocker_cmd;
    mydocker_cmd = getenv("mydocker_cmd");
    if(mydocker_cmd){
        //fprintf(stdout,"got mydocker_cmd=%s\n",mydocker_cmd);
    }else{
        //fprintf(stdout,"missing mydocker cmd env skip nsenter\n");
        return;
    }
    char nspath[1024];
    char *namespaces[] = {"ipc","uts","net","pid","mnt"};
    for(int i=0; i<5; i++){
        sprintf(nspath,"/proc/%s/ns/%s",mydocker_pid,namespaces[i]);
        int fd = open(nspath,O_RDONLY);
        if(setns(fd,0) == -1){
            fprintf(stderr,"setns %s namespace failed:%s\n",namespaces[i],strerror(errno));
        }else{
            // fprintf(stderr,"setns %s namespace succeeded\n",namespaces[i]);
        }
        close(fd);
    }
	fprintf(stderr,"setns namespace succeeded\n");
    int res = system(mydocker_cmd);
    exit(0);
    return;
}
*/
import "C"