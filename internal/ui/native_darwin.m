//go:build darwin && cgo
#import "darwin_desktop.h"
#include <stdlib.h>
#include <string.h>
#include "native_darwin.h"

static CPDesktop *desktop;
static NSString *str(const char *s) { return s?[NSString stringWithUTF8String:s]:@""; }
static NSArray *lines(const char *s) { return [str(s) componentsSeparatedByString:@"\n"]; }

void clipare_init(void) { @autoreleasepool {
 [NSApplication sharedApplication];[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
 desktop=[CPDesktop new];[desktop build];[NSApp finishLaunching];
} }
int clipare_poll(void) { @autoreleasepool {
 [[NSRunLoop currentRunLoop]runMode:NSDefaultRunLoopMode beforeDate:[NSDate dateWithTimeIntervalSinceNow:0.001]];
 NSEvent *event;while((event=[NSApp nextEventMatchingMask:NSEventMaskAny untilDate:NSDate.distantPast inMode:NSDefaultRunLoopMode dequeue:YES]))[NSApp sendEvent:event];
 [NSApp updateWindows];if(!desktop.events.count)return 0;int result=[desktop.events[0]intValue];[desktop.events removeObjectAtIndex:0];return result;
} }
int clipare_view(void) {return (int)desktop.view;}
int clipare_disclosures(void) {return (desktop.additionalExpanded?1:0)|(desktop.advancedExpanded?2:0);}
void clipare_show(void) { @autoreleasepool {[desktop show:desktop.view];} }
void clipare_set(int i,const char *text) { @autoreleasepool {if(i>=0&&i<11&&desktop.fields[i]!=NSNull.null)[desktop.fields[i]setStringValue:str(text)];} }
char *clipare_get(int i) { @autoreleasepool {if(i<0||i>=11||desktop.fields[i]==NSNull.null)return strdup("");return strdup([[desktop.fields[i]stringValue]UTF8String]);} }
void clipare_auto(int enabled) {desktop.autostart.state=enabled?NSControlStateValueOn:NSControlStateValueOff;}
int clipare_is_auto(void) {return desktop.autostart.state==NSControlStateValueOn;}
void clipare_preferences(int autostart,int updates) {clipare_auto(autostart);desktop.updates.state=updates?NSControlStateValueOn:NSControlStateValueOff;}
void clipare_addresses(const char *s) { @autoreleasepool {NSComboBox *box=desktop.fields[2];[box removeAllItems];for(NSString *line in lines(s))if(line.length)[box addItemWithObjectValue:line];} }
void clipare_peers(const char *s) { @autoreleasepool {[desktop.peers removeAllObjects];for(NSString *line in lines(s))if(line.length)[desktop.peers addObject:@{@"name":line,@"detail":@"Не в сети",@"online":@NO}];desktop.devicesEmpty=!desktop.peers.count;desktop.canRemove=desktop.compactVisible=desktop.listVisible=desktop.peers.count>0;[desktop refreshDevices];} }
int clipare_selected(void) {return (int)desktop.peerTable.selectedRow;}
void clipare_status(const char *status,int enabled,const char *s) { @autoreleasepool {
 [desktop.peers removeAllObjects];for(NSString *line in lines(s)){
  if(![line hasPrefix:@"● "]&&![line hasPrefix:@"○ "])continue;
  BOOL online=[line hasPrefix:@"● "];[desktop.peers addObject:@{@"name":[line substringFromIndex:2],@"detail":online?@"Подключено":@"Не в сети",@"online":@(online)}];
 }[desktop refreshDevices];[desktop refreshTray];
} }
void clipare_sync_status(int state,const char *message,const char *reason) { @autoreleasepool {if(desktop.syncState==state&&[desktop.statusMessage isEqual:str(message)]&&[desktop.statusReason isEqual:str(reason)])return;desktop.syncState=state;desktop.statusMessage=str(message);desktop.statusReason=str(reason);[desktop refreshTray];} }
void clipare_device_rows(const char *json,int empty,int list,int remove,int compact) { @autoreleasepool {
 NSData *data=[str(json)dataUsingEncoding:NSUTF8StringEncoding];id rows=[NSJSONSerialization JSONObjectWithData:data options:0 error:nil];if(![rows isKindOfClass:NSArray.class])return;
 if([desktop.peers isEqual:rows]&&desktop.devicesEmpty==empty&&desktop.listVisible==list&&desktop.compactVisible==compact)return;
 desktop.devicesEmpty=empty;desktop.listVisible=list;desktop.canRemove=remove;desktop.compactVisible=compact;desktop.peers=[NSMutableArray arrayWithArray:rows];[desktop refreshDevices];[desktop refreshTray];
} }
void clipare_alert(const char *text) { @autoreleasepool {
 if(desktop.window.attachedSheet)return;NSAlert *alert=[[[NSAlert alloc]init]autorelease];alert.messageText=@"Clipare";alert.informativeText=str(text);[alert addButtonWithTitle:@"Понятно"];
 [desktop.window makeKeyAndOrderFront:nil];[alert beginSheetModalForWindow:desktop.window completionHandler:nil];
} }
void clipare_actions(unsigned long long mask) {for(NSButton *button in desktop.actionButtons)button.enabled=(mask&(1ULL<<button.tag))!=0;}
void clipare_close(void) { @autoreleasepool {
 desktop.closed=YES;[desktop clearPairNotification];if([NSBundle.mainBundle.bundleIdentifier isEqualToString:@"io.clipare.app"])[UNUserNotificationCenter currentNotificationCenter].delegate=nil;
 if(desktop.window.attachedSheet)[desktop.window endSheet:desktop.window.attachedSheet];
 [[NSStatusBar systemStatusBar]removeStatusItem:desktop.tray];desktop.window.delegate=nil;[desktop.window close];[desktop release];desktop=nil;
} }
void clipare_discovered(const char *s,const char *status) { @autoreleasepool {
 [desktop.found removeAllObjects];for(NSString *line in lines(s)){if(!line.length)continue;NSArray *parts=[line componentsSeparatedByString:@" — "];[desktop.found addObject:@{@"name":parts[0],@"detail":parts.count>1?parts[1]:@"",@"online":@YES}];}
 for(NSLayoutConstraint *constraint in desktop.foundList.constraints)if([constraint.identifier isEqualToString:@"deviceListHeight"])constraint.constant=MIN(MAX(desktop.found.count*52+4,56),160);
 [desktop.foundTable reloadData];desktop.foundList.hidden=!desktop.found.count;desktop.emptyFound.hidden=desktop.found.count>0;desktop.connect.enabled=desktop.found.count>0;desktop.discoveryStatus.stringValue=str(status);[desktop show:CPDiscovery];
} }
int clipare_discovered_selected(void) {return (int)desktop.foundTable.selectedRow;}
void clipare_pair(const char *name,const char *sas,int mode) { @autoreleasepool {
 if(mode==1&&!desktop.pairIncoming)[desktop notifyPair:str(name)];
 desktop.pairIncoming=mode==1;desktop.pairName.stringValue=str(name);desktop.sas.stringValue=str(sas);
 desktop.pairHelp.stringValue=mode==2?@"Сравните коды на обоих компьютерах и нажмите «Код совпадает». На другом устройстве также разрешите подключение.":mode==1?@"Сравните коды на обоих компьютерах. Если они отличаются — отклоните подключение.":@"Сравните код на другом компьютере и разрешите подключение там. Ожидание подтверждения…";
 desktop.approve.hidden=mode==0;desktop.approve.title=mode==2?@"Код совпадает":@"Разрешить";desktop.approve.tag=mode==2?18:14;desktop.reject.title=mode==1?@"Отклонить":@"Отменить";desktop.reject.tag=mode==1?15:16;[desktop show:CPPairing];
} }
void clipare_pair_close(void) { @autoreleasepool {[desktop clearPairNotification];if(desktop.noticeOrigin==CPPairing)desktop.noticeOrigin=CPHome;if(desktop.view==CPPairing)[desktop show:CPHome];} }
void clipare_update_settings(const char *version,int enabled) { @autoreleasepool {desktop.version.stringValue=[@"Версия: "stringByAppendingString:str(version)];desktop.updates.state=enabled?NSControlStateValueOn:NSControlStateValueOff;} }
int clipare_update_enabled(void) {return desktop.updates.state==NSControlStateValueOn;}
void clipare_update_prompt(const char *text,const char *primary,const char *dismiss,int action) { @autoreleasepool {
 desktop.updateText.stringValue=str(text);desktop.install.title=str(primary);desktop.install.tag=action;desktop.install.hidden=!str(primary).length;desktop.dismiss.title=str(dismiss);desktop.dismiss.hidden=!str(dismiss).length;
 desktop.install.keyEquivalent=str(primary).length?@"\r":@"";desktop.dismiss.keyEquivalent=!str(primary).length&&str(dismiss).length?@"\r":@"\033";[desktop show:CPUpdate];
} }
void clipare_update_close(void) { @autoreleasepool {if(desktop.noticeOrigin==CPUpdate)desktop.noticeOrigin=CPHome;if(desktop.view==CPUpdate)[desktop show:CPHome];} }
