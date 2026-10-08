//go:build darwin && cgo
#import <AppKit/AppKit.h>
#include <stdlib.h>
#include <string.h>
#include "native_darwin.h"

static NSWindow *window;
static NSStatusItem *tray;
static id fields[11];
static NSButton *autostart;
static NSTableView *table;
static NSMutableArray *rows;
static NSMutableArray *events;
static NSString *stateText;
static BOOL syncEnabled;
static NSTextField *connectionStatus;
static NSWindow *homeWindow,*discoveryWindow,*pairWindow;
static NSTextField *homeName,*homeStatus,*discoveryStatus,*pairName,*pairCode,*pairHelp;
static NSButton *homeAuto,*approveButton,*rejectButton;
static NSTableView *homeTable,*foundTable;
static NSMutableArray *foundRows;
static NSMutableArray *homeRows;
static BOOL pairIncoming;
static NSWindow *updateWindow;
static NSButton *updateCheck,*updateInstall,*updateDismiss;
static NSTextField *updateVersion,*updateText;
@interface ClipareDelegate : NSObject <NSTableViewDataSource,NSTableViewDelegate,NSWindowDelegate>
-(void)action:(id)sender;
@end
static ClipareDelegate *delegate;
@implementation ClipareDelegate
-(void)action:(id)sender {
 if([sender tag]==30){[homeWindow orderOut:nil];[window makeKeyAndOrderFront:nil];return;}
 if([sender tag]==1){if([sender window]==homeWindow){[fields[0] setStringValue:homeName.stringValue];autostart.state=homeAuto.state;}else{homeName.stringValue=[fields[0] stringValue];homeAuto.state=autostart.state;}}
 [events addObject:@([sender tag])];
}
-(NSInteger)numberOfRowsInTableView:(NSTableView *)view {return view==foundTable?foundRows.count:view==homeTable?homeRows.count:rows.count;}
-(id)tableView:(NSTableView *)view objectValueForTableColumn:(NSTableColumn *)column row:(NSInteger)row {return view==foundTable?foundRows[row]:view==homeTable?homeRows[row]:rows[row];}
-(void)tableViewSelectionDidChange:(NSNotification *)n { if(n.object!=foundTable)[events addObject:@10]; }
-(BOOL)windowShouldClose:(NSWindow *)w { if(w==discoveryWindow)[events addObject:@17];if(w==pairWindow)[events addObject:@(pairIncoming?15:16)];[w orderOut:nil]; return NO; }
@end
static NSString *str(const char *s) { return s ? [NSString stringWithUTF8String:s] : @""; }
static void label(NSView *root,NSString *text,CGFloat x,CGFloat y,CGFloat width) {
 NSTextField *v=[NSTextField labelWithString:text]; v.frame=NSMakeRect(x,y,width,20); [root addSubview:v];
}
static void input(NSView *root,int index,CGFloat x,CGFloat y,CGFloat width,BOOL secure) {
 NSTextField *v=secure ? [[NSSecureTextField alloc] initWithFrame:NSMakeRect(x,y,width,26)] : [[NSTextField alloc] initWithFrame:NSMakeRect(x,y,width,26)];
 fields[index]=v; [root addSubview:v]; [v release];
}
static void button(NSView *root,NSString *text,CGFloat x,CGFloat y,CGFloat width,int tag) {
 NSButton *v=[NSButton buttonWithTitle:text target:delegate action:@selector(action:)]; v.tag=tag; v.frame=NSMakeRect(x,y,width,30); [root addSubview:v];
}
static NSWindow *panel(NSString *title,CGFloat width,CGFloat height){NSWindow *w=[[NSWindow alloc] initWithContentRect:NSMakeRect(0,0,width,height) styleMask:NSWindowStyleMaskTitled|NSWindowStyleMaskClosable backing:NSBackingStoreBuffered defer:NO];w.title=title;w.releasedWhenClosed=NO;w.delegate=delegate;[w center];return w;}
static NSTableView *device_table(NSView *root,CGFloat x,CGFloat y,CGFloat width,CGFloat height){NSScrollView *scroll=[[NSScrollView alloc] initWithFrame:NSMakeRect(x,y,width,height)];scroll.hasVerticalScroller=YES;scroll.borderType=NSBezelBorder;NSTableView *t=[[NSTableView alloc] initWithFrame:scroll.bounds];NSTableColumn *column=[[NSTableColumn alloc] initWithIdentifier:@"device"];column.width=width-20;column.editable=NO;[t addTableColumn:column];[column release];t.headerView=nil;t.dataSource=delegate;t.delegate=delegate;scroll.documentView=t;[root addSubview:scroll];[scroll release];return t;}
static void install_home(void){
 homeWindow=panel(@"Clipare",480,594);NSView *root=homeWindow.contentView;
 // Preserve the established home layout; updates occupy a separate lower block.
 NSView *deviceRoot=[[NSView alloc] initWithFrame:NSMakeRect(0,144,480,450)];[root addSubview:deviceRoot];root=deviceRoot;
 label(root,@"Этот компьютер",24,406,420);homeName=[[NSTextField alloc] initWithFrame:NSMakeRect(24,368,432,28)];[root addSubview:homeName];
 homeStatus=[NSTextField labelWithString:@"Ожидание NetBird"];homeStatus.frame=NSMakeRect(24,322,432,34);homeStatus.lineBreakMode=NSLineBreakByWordWrapping;[root addSubview:homeStatus];
 label(root,@"Устройства",24,284,420);homeTable=device_table(root,24,140,432,136);
 button(root,@"+ Добавить устройство",24,96,254,11);button(root,@"Удалить",334,96,122,9);
 homeAuto=[NSButton checkboxWithTitle:@"Запускать при входе в систему" target:nil action:nil];homeAuto.frame=NSMakeRect(24,60,420,26);[root addSubview:homeAuto];
 button(root,@"Дополнительно…",24,16,190,30);button(root,@"Сохранить",320,16,136,1);
 root=homeWindow.contentView;
 label(root,@"Обновления",24,110,432);
 updateCheck=[NSButton checkboxWithTitle:@"Автоматически проверять обновления" target:delegate action:@selector(action:)];updateCheck.tag=22;updateCheck.frame=NSMakeRect(24,72,432,26);[root addSubview:updateCheck];
 updateVersion=[NSTextField labelWithString:@""];updateVersion.frame=NSMakeRect(24,30,230,22);[root addSubview:updateVersion];button(root,@"Проверить обновления",254,24,202,19);
 updateWindow=panel(@"Обновление Clipare",480,260);root=updateWindow.contentView;
 updateText=[NSTextField labelWithString:@""];updateText.frame=NSMakeRect(24,82,432,150);updateText.lineBreakMode=NSLineBreakByWordWrapping;[root addSubview:updateText];
 updateInstall=[NSButton buttonWithTitle:@"Обновить" target:delegate action:@selector(action:)];updateInstall.tag=20;updateInstall.frame=NSMakeRect(292,24,164,32);[root addSubview:updateInstall];
 updateDismiss=[NSButton buttonWithTitle:@"Понятно" target:delegate action:@selector(action:)];updateDismiss.tag=21;updateDismiss.frame=NSMakeRect(24,24,164,32);updateDismiss.keyEquivalent=@"\033";[root addSubview:updateDismiss];
 discoveryWindow=panel(@"Добавить устройство",480,380);root=discoveryWindow.contentView;
 label(root,@"Найденные устройства",24,336,432);foundTable=device_table(root,24,144,432,180);
 discoveryStatus=[NSTextField labelWithString:@"Поиск устройств…"];discoveryStatus.frame=NSMakeRect(24,92,432,44);discoveryStatus.lineBreakMode=NSLineBreakByWordWrapping;[root addSubview:discoveryStatus];
 button(root,@"Обновить",24,50,126,12);button(root,@"Подключить",302,50,154,13);button(root,@"Не нашли? Добавить по коду…",24,10,310,30);
 pairWindow=panel(@"Подключение устройства",480,300);root=pairWindow.contentView;
 pairName=[NSTextField labelWithString:@""];pairName.frame=NSMakeRect(24,240,432,36);pairName.font=[NSFont systemFontOfSize:17 weight:NSFontWeightMedium];[root addSubview:pairName];
 pairCode=[NSTextField labelWithString:@""];pairCode.frame=NSMakeRect(24,156,432,56);pairCode.font=[NSFont monospacedDigitSystemFontOfSize:38 weight:NSFontWeightMedium];pairCode.alignment=NSTextAlignmentCenter;[root addSubview:pairCode];
 pairHelp=[NSTextField labelWithString:@""];pairHelp.frame=NSMakeRect(24,78,432,60);pairHelp.lineBreakMode=NSLineBreakByWordWrapping;[root addSubview:pairHelp];
 approveButton=[NSButton buttonWithTitle:@"Разрешить" target:delegate action:@selector(action:)];approveButton.tag=14;approveButton.frame=NSMakeRect(292,20,164,32);[root addSubview:approveButton];
 rejectButton=[NSButton buttonWithTitle:@"Отклонить" target:delegate action:@selector(action:)];rejectButton.tag=15;rejectButton.frame=NSMakeRect(24,20,164,32);[root addSubview:rejectButton];
}
// AppKit routes Command shortcuts through the application menu to the field
// editor. A status-bar menu alone does not provide the standard editing actions.
static void install_edit_menu(void) {
 NSMenu *main=[[NSMenu alloc] initWithTitle:@"Clipare"];
 NSMenuItem *appItem=[[NSMenuItem alloc] initWithTitle:@"Clipare" action:nil keyEquivalent:@""];
 NSMenu *appMenu=[[NSMenu alloc] initWithTitle:@"Clipare"];
 NSMenuItem *quit=[[NSMenuItem alloc] initWithTitle:@"Выйти из Clipare" action:@selector(action:) keyEquivalent:@"q"];
 quit.target=delegate;quit.tag=4;[appMenu addItem:quit];[quit release];appItem.submenu=appMenu;[appMenu release];[main addItem:appItem];[appItem release];
 NSMenuItem *editItem=[[NSMenuItem alloc] initWithTitle:@"Правка" action:nil keyEquivalent:@""];
 NSMenu *edit=[[NSMenu alloc] initWithTitle:@"Правка"];
 NSArray *titles=@[@"Отменить",@"Вырезать",@"Копировать",@"Вставить",@"Выделить всё"];
 NSArray *keys=@[@"z",@"x",@"c",@"v",@"a"];
 SEL actions[]={@selector(undo:),@selector(cut:),@selector(copy:),@selector(paste:),@selector(selectAll:)};
 for(NSUInteger i=0;i<titles.count;i++){
  NSMenuItem *item=[[NSMenuItem alloc] initWithTitle:titles[i] action:actions[i] keyEquivalent:keys[i]];
  item.keyEquivalentModifierMask=NSEventModifierFlagCommand;
  [edit addItem:item];[item release];
 }
 editItem.submenu=edit;[edit release];[main addItem:editItem];[editItem release];[NSApp setMainMenu:main];[main release];
}
void clipare_init(void) { @autoreleasepool {
 [NSApplication sharedApplication]; [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
 delegate=[ClipareDelegate new]; rows=[NSMutableArray new];foundRows=[NSMutableArray new];homeRows=[NSMutableArray new];events=[NSMutableArray new];stateText=[@"Clipare" copy];
 install_edit_menu();
 tray=[[[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength] retain];
 tray.button.title=@"⇄"; tray.button.font=[NSFont systemFontOfSize:20 weight:NSFontWeightMedium]; tray.button.toolTip=@"Clipare";
 window=[[NSWindow alloc] initWithContentRect:NSMakeRect(0,0,760,620) styleMask:NSWindowStyleMaskTitled|NSWindowStyleMaskClosable|NSWindowStyleMaskMiniaturizable backing:NSBackingStoreBuffered defer:NO];
 window.title=@"Настройки Clipare";window.releasedWhenClosed=NO;window.delegate=delegate;[window center];
 NSView *root=window.contentView;
 label(root,@"Этот компьютер",24,576,320);
 label(root,@"Имя устройства",24,548,320);input(root,0,24,519,320,NO);
 label(root,@"ID устройства (создаётся автоматически)",24,489,320);input(root,1,24,460,320,NO);[fields[1] setEditable:NO];
 label(root,@"NetBird IP для подключения",24,430,320);
 NSComboBox *addresses=[[NSComboBox alloc] initWithFrame:NSMakeRect(24,401,320,26)];fields[2]=addresses;[root addSubview:addresses];[addresses release];
 label(root,@"Порт",24,371,320);input(root,3,24,342,320,NO);
 label(root,@"Legacy общий ключ",24,312,320);input(root,4,24,283,320,YES);
 button(root,@"Создать новый ключ",24,242,190,5);
 button(root,@"Скопировать код подключения",24,198,320,6);
 autostart=[NSButton checkboxWithTitle:@"Запускать при входе в систему" target:nil action:nil];autostart.frame=NSMakeRect(24,154,320,26);[root addSubview:autostart];
 connectionStatus=[NSTextField labelWithString:@"Настройте подключение"];connectionStatus.frame=NSMakeRect(24,98,320,48);connectionStatus.font=[NSFont systemFontOfSize:12];connectionStatus.lineBreakMode=NSLineBreakByWordWrapping;[root addSubview:connectionStatus];
 label(root,@"Другие компьютеры",390,576,340);
 NSScrollView *scroll=[[NSScrollView alloc] initWithFrame:NSMakeRect(390,433,340,135)];scroll.hasVerticalScroller=YES;scroll.borderType=NSBezelBorder;
 table=[[NSTableView alloc] initWithFrame:scroll.bounds];NSTableColumn *column=[[NSTableColumn alloc] initWithIdentifier:@"peer"];column.title=@"Устройства";column.width=320;column.editable=NO;[table addTableColumn:column];[column release];table.headerView=nil;table.dataSource=delegate;table.delegate=delegate;scroll.documentView=table;[root addSubview:scroll];[scroll release];
 label(root,@"Имя",390,405,150);input(root,6,390,376,340,NO);
 label(root,@"ID другого устройства",390,346,340);input(root,7,390,317,340,NO);
 label(root,@"IP или имя NetBird",390,287,230);input(root,8,390,258,235,NO);
 label(root,@"Порт",635,287,95);input(root,9,635,258,95,NO);
 button(root,@"Добавить / изменить",390,215,220,8);button(root,@"Удалить",618,215,112,9);
 label(root,@"Код подключения другого компьютера",390,181,340);input(root,10,390,152,340,YES);
 button(root,@"Добавить по коду",390,111,220,7);
 NSTextField *fingerprint=[NSTextField labelWithString:@""];fingerprint.frame=NSMakeRect(24,76,710,20);fields[5]=fingerprint;[root addSubview:fingerprint];
 label(root,@"Код содержит общий ключ. Передавайте его только доверенным устройствам.",24,52,710);
 button(root,@"Сохранить настройки",500,10,230,1);
 install_home();[NSApp finishLaunching];
} }
int clipare_poll(void) { @autoreleasepool {

 [[NSRunLoop currentRunLoop] runMode:NSDefaultRunLoopMode beforeDate:[NSDate dateWithTimeIntervalSinceNow:0.001]];
 NSEvent *event;while((event=[NSApp nextEventMatchingMask:NSEventMaskAny untilDate:[NSDate distantPast] inMode:NSDefaultRunLoopMode dequeue:YES])){[NSApp sendEvent:event];}
 [NSApp updateWindows];if(events.count==0)return 0;int result=[events[0] intValue];[events removeObjectAtIndex:0];return result;
} }
void clipare_show(void) { @autoreleasepool {homeName.stringValue=[fields[0] stringValue];homeAuto.state=autostart.state;[homeTable reloadData];if(window.visible)[window makeKeyAndOrderFront:nil];else [homeWindow makeKeyAndOrderFront:nil];[NSApp activateIgnoringOtherApps:YES]; } }
void clipare_set(int i,const char *text) { @autoreleasepool {if(i>=0&&i<11&&fields[i]) [fields[i] setStringValue:str(text)];} }
char *clipare_get(int i) { @autoreleasepool {if(i<0||i>=11||!fields[i])return strdup("");if(i==0&&homeWindow.visible&&!window.visible)return strdup([homeName.stringValue UTF8String]);return strdup([[fields[i] stringValue] UTF8String]);} }
void clipare_auto(int enabled) { autostart.state=enabled?NSControlStateValueOn:NSControlStateValueOff; }
int clipare_is_auto(void) { return (homeWindow.visible&&!window.visible?homeAuto.state:autostart.state)==NSControlStateValueOn; }
void clipare_addresses(const char *lines) { @autoreleasepool {NSComboBox *box=fields[2];[box removeAllItems];for(NSString *s in [str(lines) componentsSeparatedByString:@"\n"]){if(s.length)[box addItemWithObjectValue:s];}} }
void clipare_peers(const char *lines) { @autoreleasepool {[rows removeAllObjects];for(NSString *s in [str(lines) componentsSeparatedByString:@"\n"]){if(s.length)[rows addObject:s];}[table reloadData];[homeTable reloadData];[table deselectAll:nil];} }
int clipare_selected(void) { return (int)(window.visible?table.selectedRow:homeTable.selectedRow); }
void clipare_status(const char *status,int enabled,const char *peers) { @autoreleasepool {
 [stateText release];stateText=[str(status) copy];syncEnabled=enabled;
 connectionStatus.stringValue=stateText;
 homeStatus.stringValue=stateText;
 [homeRows removeAllObjects];for(NSString *s in [str(peers) componentsSeparatedByString:@"\n"]){if(s.length)[homeRows addObject:s];}[homeTable reloadData];
 NSMenu *menu=[[NSMenu alloc] initWithTitle:@"Clipare"];
 NSMenuItem *header=[[NSMenuItem alloc] initWithTitle:@"Clipare" action:nil keyEquivalent:@""];[menu addItem:header];[header release];
 NSMenuItem *state=[[NSMenuItem alloc] initWithTitle:stateText action:nil keyEquivalent:@""];[menu addItem:state];[state release];[menu addItem:[NSMenuItem separatorItem]];
 for(NSString *s in [str(peers) componentsSeparatedByString:@"\n"]){if(!s.length)continue;NSMenuItem *item=[[NSMenuItem alloc] initWithTitle:s action:nil keyEquivalent:@""];[menu addItem:item];[item release];}
 [menu addItem:[NSMenuItem separatorItem]];
 NSArray *titles=@[enabled?@"Приостановить синхронизацию":@"Возобновить синхронизацию",@"Настройки…",@"Выйти"];
 int tags[]={3,2,4};for(int i=0;i<3;i++){NSMenuItem *item=[[NSMenuItem alloc] initWithTitle:titles[i] action:@selector(action:) keyEquivalent:@""];item.target=delegate;item.tag=tags[i];[menu addItem:item];[item release];}
 NSMenuItem *check=[[NSMenuItem alloc] initWithTitle:@"Проверить обновления" action:@selector(action:) keyEquivalent:@""];check.target=delegate;check.tag=19;[menu insertItem:check atIndex:menu.numberOfItems-1];[check release];
 tray.menu=menu;[menu release];tray.button.toolTip=stateText;
} }
void clipare_alert(const char *text) { @autoreleasepool {NSAlert *a=[NSAlert new];a.messageText=@"Clipare";a.informativeText=str(text);[a addButtonWithTitle:@"OK"];[window makeKeyAndOrderFront:nil];[NSApp activateIgnoringOtherApps:YES];[a beginSheetModalForWindow:window completionHandler:^(NSModalResponse response){[a release];}];} }
void clipare_close(void) { @autoreleasepool {if(window.attachedSheet)[window endSheet:window.attachedSheet];[[NSStatusBar systemStatusBar] removeStatusItem:tray];[window orderOut:nil];[homeWindow orderOut:nil];[discoveryWindow orderOut:nil];[pairWindow orderOut:nil];[updateWindow orderOut:nil];} }
void clipare_discovered(const char *lines,const char *status){@autoreleasepool{[foundRows removeAllObjects];for(NSString *s in [str(lines) componentsSeparatedByString:@"\n"]){if(s.length)[foundRows addObject:s];}[foundTable reloadData];discoveryStatus.stringValue=str(status);[discoveryWindow makeKeyAndOrderFront:nil];[NSApp activateIgnoringOtherApps:YES];}}
int clipare_discovered_selected(void){return (int)foundTable.selectedRow;}
void clipare_pair(const char *name,const char *sas,int mode){@autoreleasepool{pairIncoming=mode==1;pairName.stringValue=str(name);pairCode.stringValue=str(sas);pairHelp.stringValue=mode==2?@"Сравните коды на обоих компьютерах и нажмите «Код совпадает». На другом устройстве также разрешите подключение.":pairIncoming?@"Это устройство хочет подключиться. Сравните коды на обоих компьютерах. Если они отличаются — отклоните подключение.":@"Сравните код на другом компьютере и разрешите подключение там. Ожидание подтверждения…";approveButton.hidden=mode==0;approveButton.title=mode==2?@"Код совпадает":@"Разрешить";approveButton.tag=mode==2?18:14;rejectButton.title=pairIncoming?@"Отклонить":@"Отменить";rejectButton.tag=pairIncoming?15:16;[pairWindow makeKeyAndOrderFront:nil];[NSApp activateIgnoringOtherApps:YES];}}
void clipare_pair_close(void){[pairWindow orderOut:nil];}
void clipare_update_settings(const char *version,int enabled){@autoreleasepool{updateVersion.stringValue=[@"Версия: " stringByAppendingString:str(version)];updateCheck.state=enabled?NSControlStateValueOn:NSControlStateValueOff;}}
int clipare_update_enabled(void){return updateCheck.state==NSControlStateValueOn;}
void clipare_update_prompt(const char *text,const char *primary,const char *dismiss,int action){@autoreleasepool{updateText.stringValue=str(text);updateInstall.title=str(primary);updateInstall.tag=action;updateInstall.hidden=!str(primary).length;updateDismiss.title=str(dismiss);updateDismiss.hidden=!str(dismiss).length;updateDismiss.frame=NSMakeRect(str(primary).length?24:292,24,164,32);updateInstall.keyEquivalent=str(primary).length?@"\r":@"";if(!str(primary).length&&str(dismiss).length)updateDismiss.keyEquivalent=@"\r";else updateDismiss.keyEquivalent=@"\033";[updateWindow makeKeyAndOrderFront:nil];[NSApp activateIgnoringOtherApps:YES];}}
void clipare_update_close(void){[updateWindow orderOut:nil];}
