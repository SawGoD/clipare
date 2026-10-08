// A synthetic UI fixture: no clipboard access, config files, networking or keys.
#import <AppKit/AppKit.h>
#include <assert.h>
#include "../../internal/ui/native_darwin.m"
int main(int argc,const char **argv){@autoreleasepool{
 clipare_init();clipare_set(0,"MacBook — тест интерфейса");clipare_peers("Desktop-PC\nLaptop");clipare_status("Синхронизация включена",1,"● Desktop-PC\n○ Laptop");clipare_show();
 assert(homeWindow.visible);assert(!window.visible);assert(homeName.stringValue.length>0);
 if(argc>1 && !strcmp(argv[1],"discovery")){clipare_discovered("Desktop-PC — 100.64.0.2\nLaptop — 100.64.0.3","Выберите устройство и нажмите «Подключить»");assert(foundRows.count==2);}
 if(argc>1 && !strcmp(argv[1],"pair")){clipare_pair("Desktop-PC хочет подключиться","482 731",1);assert(!approveButton.hidden);assert(rejectButton.tag==15);assert([pairCode.stringValue isEqualToString:@"482 731"]);}
 [NSApp updateWindows];
 NSWindow *target=argc>1&&!strcmp(argv[1],"pair")?pairWindow:argc>1&&!strcmp(argv[1],"discovery")?discoveryWindow:homeWindow;
 NSBitmapImageRep *bitmap=[target.contentView bitmapImageRepForCachingDisplayInRect:target.contentView.bounds];[target.contentView cacheDisplayInRect:target.contentView.bounds toBitmapImageRep:bitmap];
 [[bitmap representationUsingType:NSBitmapImageFileTypePNG properties:@{}] writeToFile:@"/private/tmp/clipare-pairing-ui.png" atomically:YES];
 NSDate *until=[NSDate dateWithTimeIntervalSinceNow:argc>1&&!strcmp(argv[1],"verify")?0.1:40];while([until timeIntervalSinceNow]>0){int action=clipare_poll();if(action==11||action==12)clipare_discovered("Desktop-PC — 100.64.0.2\nLaptop — 100.64.0.3","Выберите устройство и нажмите «Подключить»");if(action==13)clipare_pair("Проверьте код на Desktop-PC","482 731",0);if(action==14||action==15||action==16)clipare_pair_close();if(action==2)clipare_show();if(action==4)break;[NSThread sleepForTimeInterval:0.02];}
 clipare_pair("Desktop-PC","482 731",0);assert(approveButton.hidden);assert(rejectButton.tag==16);clipare_pair_close();clipare_close();return 0;
}}
