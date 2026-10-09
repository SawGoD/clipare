// Synthetic native UI fixture: no clipboard, networking, keys or user config.
#import <AppKit/AppKit.h>
#include <assert.h>
#include "../../internal/ui/native_darwin.m"

@interface NSView (MaterialFixture)
-(void)applyReducedTransparency:(BOOL)value;
@end
static void opaqueCards(NSView *view) {
 if([view respondsToSelector:@selector(applyReducedTransparency:)]){
  [view applyReducedTransparency:YES];assert([view valueForKey:@"effect"]==nil);
  NSColor *background=[[NSColor colorWithCGColor:view.layer.backgroundColor]colorUsingColorSpace:NSColorSpace.genericRGBColorSpace];
  BOOL light=[[view.effectiveAppearance bestMatchFromAppearancesWithNames:@[NSAppearanceNameAqua,NSAppearanceNameDarkAqua]]isEqualToString:NSAppearanceNameAqua];
  assert(light?background.redComponent>0.5:background.redComponent<0.5);
 }
 for(NSView *child in view.subviews)opaqueCards(child);
}
static void fallbackCards(NSView *view) {
 if([view respondsToSelector:@selector(applyReducedTransparency:)]){
  [view setValue:@YES forKey:@"forceFallback"];[view applyReducedTransparency:NO];assert([[view valueForKey:@"effect"]isKindOfClass:NSVisualEffectView.class]);
 }
 for(NSView *child in view.subviews)fallbackCards(child);
}
static void capture(NSString *name) {
 for(int i=0;i<10;i++){clipare_poll();}
 [desktop resizeDocument];[desktop.window.contentView layoutSubtreeIfNeeded];[NSApp updateWindows];
 NSView *view=desktop.window.contentView;
 NSBitmapImageRep *image=[view bitmapImageRepForCachingDisplayInRect:view.bounds];
 [view cacheDisplayInRect:view.bounds toBitmapImageRep:image];
 NSString *dir=@"/private/tmp/clipare-native-ux";
 [[NSFileManager defaultManager]createDirectoryAtPath:dir withIntermediateDirectories:YES attributes:nil error:nil];
 [[image representationUsingType:NSBitmapImageFileTypePNG properties:@{}]writeToFile:[dir stringByAppendingPathComponent:[name stringByAppendingString:@".png"]] atomically:YES];
}
int main(int argc,const char **argv) { @autoreleasepool {
 clipare_init();NSWindow *original=desktop.window;
 // Match production startup: an unchanged empty payload skips the cache refresh.
 clipare_device_rows("[]",1,0,0,0);
 assert(desktop.peerList.hidden);assert(desktop.compactAdd.hidden);assert(!desktop.emptyDevices.hidden);
 clipare_set(0,"MacBook — тест интерфейса");clipare_set(1,"synthetic-device");clipare_set(2,"100.64.0.1");clipare_set(3,"45873");clipare_set(5,"Public key SHA-256: synthetic fingerprint");clipare_show();
 assert(desktop.view==CPHome);assert(!desktop.emptyDevices.hidden);assert(desktop.peerList.hidden);assert(desktop.removePeer==nil);
 assert(desktop.advancedButton.imagePosition==NSImageOnly&&desktop.advancedButton.toolTip.length>0);
 assert(desktop.advancedButton.superview==desktop.pause.superview);
 NSStackView *header=(NSStackView *)desktop.pause.superview;assert([header.arrangedSubviews indexOfObject:desktop.advancedButton]==[header.arrangedSubviews indexOfObject:desktop.pause]+1);
 clipare_actions(0);for(NSButton *button in desktop.actionButtons)assert(!button.enabled);
 clipare_actions(1ULL<<7);for(NSButton *button in desktop.actionButtons)if(button.tag==7)assert(button.enabled);
 clipare_actions((1ULL<<7)|(1ULL<<25));
 for(NSButton *button in desktop.actionButtons)if(button.tag==1||button.tag==25)assert(button.hidden);
 clipare_apply_visibility((1ULL<<1)|(1ULL<<25));for(NSButton *button in desktop.actionButtons)if(button.tag==1||button.tag==25)assert(!button.hidden);
 clipare_apply_visibility(1ULL<<1);for(NSButton *button in desktop.actionButtons)if(button.tag==25)assert(button.hidden);
 clipare_apply_visibility(0);[desktop action:desktop.advancedButton];for(NSButton *button in desktop.actionButtons)if(button.tag==1)assert(button.hidden);[desktop action:desktop.advancedButton];
 clipare_update_settings("0.6.0",1);assert(clipare_update_enabled());assert(desktop.updates.tag==22);assert(desktop.additional.hidden);assert(desktop.advanced.hidden);
 capture(@"empty");
 // AppKit layout uses alignment rectangles; older button frames include
 // drawing outsets which can overlap without overlapping their bezels.
 NSRect gear=[desktop.advancedButton.superview convertRect:[desktop.advancedButton alignmentRectForFrame:desktop.advancedButton.frame] toView:desktop.window.contentView];NSRect pause=[desktop.pause.superview convertRect:[desktop.pause alignmentRectForFrame:desktop.pause.frame] toView:desktop.window.contentView];
 if(NSMinX(gear)<=NSMaxX(pause))fprintf(stderr,"gear=%s pause=%s\n",NSStringFromRect(gear).UTF8String,NSStringFromRect(pause).UTF8String);
 assert(NSMinX(gear)>NSMaxX(pause));
 assert(desktop.emptyDevices.frame.size.height<=120);
 assert(desktop.scroll.documentView.frame.size.height<=desktop.scroll.contentSize.height);
 assert(desktop.version.frame.size.height>0);
 for(NSString *appearance in @[NSAppearanceNameAqua,NSAppearanceNameDarkAqua]){
  desktop.window.appearance=[NSAppearance appearanceNamed:appearance];opaqueCards(desktop.views[@0]);capture([appearance isEqualToString:NSAppearanceNameAqua]?@"empty-opaque-light":@"empty-opaque-dark");
  assert([desktop.compactAdd isHiddenOrHasHiddenAncestor]);assert(![desktop.emptyAdd isHiddenOrHasHiddenAncestor]);
  for(NSValue *size in @[[NSValue valueWithSize:NSMakeSize(560,800)],[NSValue valueWithSize:NSMakeSize(720,900)]]){[desktop.window setContentSize:size.sizeValue];[desktop resizeDocument];assert(desktop.emptyDevices.frame.size.height<=120);}
  [desktop.window setContentSize:NSMakeSize(560,600)];[desktop resizeDocument];
 }
 desktop.window.appearance=nil;[NSWorkspace.sharedWorkspace.notificationCenter postNotificationName:NSWorkspaceAccessibilityDisplayOptionsDidChangeNotification object:nil];
 [desktop action:desktop.additionalButton];assert(!desktop.additional.hidden);assert(clipare_update_enabled());[desktop action:desktop.additionalButton];assert(desktop.additional.hidden);
 [desktop action:desktop.advancedButton];assert(!desktop.advanced.hidden);assert(desktop.window==original);capture(@"advanced");[desktop action:desktop.advancedButton];assert(desktop.advanced.hidden);
 clipare_peers("Desktop-PC\nLaptop");clipare_status("Синхронизация включена",1,"● Desktop-PC\n○ Laptop");clipare_sync_status(0,"Синхронизация включена","");
 assert(desktop.emptyDevices.hidden);assert(!desktop.peerList.hidden);assert(desktop.peers.count==2);assert(desktop.syncState==0);
 const char *rows="[{\"id\":\"a\",\"name\":\"Desktop-PC\",\"detail\":\"Подключено\",\"online\":true,\"platform\":\"windows\"},{\"id\":\"b\",\"name\":\"Laptop\",\"detail\":\"Не в сети\",\"online\":false,\"platform\":\"darwin\"}]";
 for(NSString *platform in @[@"windows",@"darwin",@"",@"unrecognized"])assert(CPPlatformIcon(platform)!=nil);
 clipare_device_rows(rows,0,1,1,1);NSMenu *unchangedMenu=desktop.tray.menu;NSMutableArray *unchangedPeers=desktop.peers;
 for(int i=0;i<100;i++){clipare_device_rows(rows,0,1,1,1);clipare_sync_status(0,"Синхронизация включена","");}
 assert(desktop.tray.menu==unchangedMenu);assert(desktop.peers==unchangedPeers);
 clipare_device_rows("[]",1,0,0,0);assert(desktop.compactAdd.hidden&&desktop.peerList.hidden&&!desktop.emptyDevices.hidden);assert(desktop.emptyDevices.frame.size.height<=120);
 clipare_device_rows(rows,0,1,1,1);assert(!desktop.compactAdd.hidden&&desktop.emptyDevices.hidden);
 for(NSString *appearance in @[NSAppearanceNameAqua,NSAppearanceNameDarkAqua]) {
  desktop.window.appearance=[NSAppearance appearanceNamed:appearance];[desktop.window.contentView setNeedsDisplay:YES];capture([appearance isEqualToString:NSAppearanceNameAqua]?@"home-light":@"home-dark");
  opaqueCards(desktop.views[@0]);capture([appearance isEqualToString:NSAppearanceNameAqua]?@"opaque-light":@"opaque-dark");
  for(NSInteger row=0;row<2;row++){NSView *cell=[desktop.peerTable viewAtColumn:0 row:row makeIfNecessary:YES];NSUInteger buttons=0;for(NSView *view in cell.subviews)if([view isKindOfClass:NSButton.class]){buttons++;NSRect bounds=[view convertRect:view.bounds toView:desktop.peerTable];if(NSMaxX(bounds)>NSMaxX(desktop.peerTable.visibleRect)+1)fprintf(stderr,"row bounds=%s viewport=%s\n",NSStringFromRect(bounds).UTF8String,NSStringFromRect(desktop.peerTable.visibleRect).UTF8String);assert(NSMaxX(bounds)<=NSMaxX(desktop.peerTable.visibleRect)+1);}assert(buttons==1);}
  [NSWorkspace.sharedWorkspace.notificationCenter postNotificationName:NSWorkspaceAccessibilityDisplayOptionsDidChangeNotification object:nil];
 }
 desktop.window.appearance=nil;
 fallbackCards(desktop.views[@0]);capture(@"material-fallback");
 clipare_sync_status(1,"Синхронизация приостановлена","");capture(@"paused");
 clipare_sync_status(2,"Синхронизация недоступна","Ожидание NetBird");assert(desktop.syncState==2);capture(@"degraded");
 clipare_discovered("Desktop-PC — 100.64.0.2\nLaptop — 100.64.0.3","Выберите устройство");assert(desktop.view==CPDiscovery);assert(desktop.found.count==2);assert(desktop.window==original);capture(@"discovery");
 assert(desktop.scroll.documentView==desktop.views[@0]);assert(desktop.deviceHome.hidden&&!desktop.deviceDiscovery.hidden);assert(![desktop.fields[0] isHiddenOrHasHiddenAncestor]);assert(![desktop.status isHiddenOrHasHiddenAncestor]);
 clipare_device_rows(rows,0,1,1,1);assert(desktop.deviceHome.hidden&&!desktop.deviceDiscovery.hidden);
 for(NSString *appearance in @[NSAppearanceNameAqua,NSAppearanceNameDarkAqua]){desktop.window.appearance=[NSAppearance appearanceNamed:appearance];opaqueCards(desktop.views[@0]);capture([appearance isEqualToString:NSAppearanceNameAqua]?@"discovery-inline-light":@"discovery-inline-dark");}
 desktop.window.appearance=nil;
 [desktop back];assert(desktop.view==CPHome);
 assert(!desktop.deviceHome.hidden&&desktop.deviceDiscovery.hidden);assert(desktop.scroll.documentView==desktop.views[@0]);
 clipare_discovered("Desktop-PC — 100.64.0.2","Выберите устройство");capture(@"discovery-one");assert(desktop.foundList.frame.size.height<=60);assert(desktop.scroll.documentView.frame.size.height<=desktop.scroll.contentSize.height);
 [desktop.events removeAllObjects];[desktop action:desktop.advancedButton];assert(desktop.view==CPHome&&desktop.advancedExpanded&&[desktop.events containsObject:@17]);[desktop action:desktop.advancedButton];
 clipare_discovered("","Автоматическое обнаружение недоступно");assert(desktop.foundList.hidden);assert(!desktop.connect.enabled);assert(desktop.discoveryStatus.hidden);assert(!desktop.emptyFound.hidden);assert([desktop.emptyFound.stringValue isEqualToString:@"Автоматическое обнаружение недоступно"]);capture(@"discovery-empty");
 clipare_discovered("","Устройства Clipare не найдены.\nУбедитесь, что NetBird и Clipare запущены на обоих устройствах.");assert(desktop.discoveryStatus.hidden);assert([desktop.emptyFound.stringValue containsString:@"обоих устройствах"]);capture(@"discovery-empty-message");
 clipare_discovered("","Поиск устройств…");assert(desktop.discoveryStatus.hidden);assert([desktop.emptyFound.stringValue isEqualToString:@"Поиск устройств…"]);
 clipare_pair("Desktop-PC хочет подключиться","482 731",1);assert(desktop.view==CPPairing);assert(desktop.reject.tag==15);assert(!desktop.approve.hidden);assert([desktop.sas.stringValue isEqualToString:@"482 731"]);capture(@"pairing");
 assert(desktop.scroll.documentView==desktop.views[@0]);assert(!desktop.devicePair.hidden&&desktop.deviceHome.hidden&&desktop.deviceDiscovery.hidden);assert(![desktop.status isHiddenOrHasHiddenAncestor]);assert(![desktop.fields[0] isHiddenOrHasHiddenAncestor]);
 clipare_device_rows(rows,0,1,1,1);assert(desktop.deviceHome.hidden&&!desktop.devicePair.hidden);
 for(NSString *appearance in @[NSAppearanceNameAqua,NSAppearanceNameDarkAqua]){desktop.window.appearance=[NSAppearance appearanceNamed:appearance];opaqueCards(desktop.views[@0]);capture([appearance isEqualToString:NSAppearanceNameAqua]?@"pairing-inline-light":@"pairing-inline-dark");}desktop.window.appearance=nil;
 clipare_alert("Сначала завершите подключение");assert(desktop.view==CPPairing);assert(desktop.window.attachedSheet);[desktop.window endSheet:desktop.window.attachedSheet];for(int i=0;i<10;i++)clipare_poll();assert(desktop.pairIncoming);
 clipare_pair_close();assert(desktop.view==CPHome);assert(desktop.window==original);
 clipare_pair("Laptop","482 731",2);assert(desktop.approve.tag==18);clipare_pair_close();
 for(NSNumber *mode in @[@0,@1,@2]){clipare_pair("Laptop","482 731",mode.intValue);[desktop.events removeAllObjects];[desktop back];assert(desktop.view==CPHome);assert([desktop.events containsObject:mode.intValue==1?@15:@16]);}
 [desktop action:desktop.advancedButton];NSButton *help=nil;NSMutableArray *pending=[NSMutableArray arrayWithObject:desktop.advanced];NSUInteger helpCount=0;
 while(pending.count){NSView *view=[pending.lastObject retain];[pending removeLastObject];if([view isKindOfClass:NSButton.class]&&[(NSButton *)view tag]==40){helpCount++;help=(NSButton *)view;assert(help.toolTip.length&&help.accessibilityLabel.length);} [pending addObjectsFromArray:view.subviews];[view release];}
 assert(helpCount==4);[desktop showHelp:help];assert(desktop.window.attachedSheet&&desktop.view==CPHome);[desktop.window endSheet:desktop.window.attachedSheet];for(int i=0;i<10;i++)clipare_poll();[desktop action:desktop.advancedButton];
 [desktop action:desktop.advancedButton];[desktop resizeDocument];NSRect technical=[desktop.advanced convertRect:desktop.advanced.bounds toView:desktop.scroll.documentView];
 [desktop.scroll.contentView scrollToPoint:NSMakePoint(0,MAX(0,technical.origin.y-12))];[desktop.scroll reflectScrolledClipView:desktop.scroll.contentView];opaqueCards(desktop.views[@0]);capture(@"advanced-fields");
 [desktop.scroll.contentView scrollToPoint:NSZeroPoint];[desktop action:desktop.advancedButton];
 clipare_update_prompt("Установлена последняя версия Clipare\n\nТекущая версия: 0.6.0","","Понятно",0);assert(desktop.view==CPUpdate);assert(desktop.install.hidden);assert(!desktop.dismiss.hidden);capture(@"update-latest");
 clipare_update_prompt("Доступна новая версия","Обновить","Позже",20);assert(desktop.install.tag==20);capture(@"update-available");
 clipare_update_prompt("Загрузка обновления…","","",0);assert(desktop.install.hidden&&desktop.dismiss.hidden);[desktop back];assert(desktop.view==CPUpdate);capture(@"update-download");
 clipare_update_prompt("Не удалось обновить Clipare","Повторить","Закрыть",19);assert(desktop.install.tag==19);capture(@"update-error");
 clipare_update_close();assert(desktop.view==CPHome);
 clipare_alert("Настройки сохранены");assert(desktop.view==CPHome);assert(desktop.window==original);assert(desktop.window.attachedSheet);[desktop.window endSheet:desktop.window.attachedSheet];for(int i=0;i<10;i++)clipare_poll();assert(desktop.view==CPHome);
 NSUInteger topLevel=0;for(NSWindow *w in NSApp.windows)if(w==original||([w.title isEqualToString:@"Clipare"]&&w.visible))topLevel++;assert(topLevel==1);
 if(argc>1&&!strcmp(argv[1],"preview")){NSDate *until=[NSDate dateWithTimeIntervalSinceNow:120];while(until.timeIntervalSinceNow>0){int event=clipare_poll();if(event==4)break;if(event==11||event==12)clipare_discovered("Desktop-PC — 100.64.0.2","Выберите устройство");if(event==13)clipare_pair("Desktop-PC","482 731",1);if(event==14||event==15||event==16)clipare_pair_close();[NSThread sleepForTimeInterval:0.02];}}
 clipare_close();puts("PASS: single-window navigation, typed status, empty state, disclosures, pairing, updates");return 0;
} }
