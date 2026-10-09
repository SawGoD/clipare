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
 clipare_set(0,"MacBook — тест интерфейса");clipare_set(1,"synthetic-device");clipare_set(2,"100.64.0.1");clipare_set(3,"45873");clipare_set(5,"Public key SHA-256: synthetic fingerprint");clipare_peers("");clipare_show();
 assert(desktop.view==CPHome);assert(!desktop.emptyDevices.hidden);assert(desktop.peerList.hidden);assert(desktop.removePeer.hidden);
 clipare_update_settings("0.6.0",1);assert(clipare_update_enabled());assert(desktop.updates.tag==22);assert(desktop.additional.hidden);assert(desktop.advanced.hidden);
 capture(@"empty");
 [desktop action:desktop.additionalButton];assert(!desktop.additional.hidden);assert(clipare_update_enabled());[desktop action:desktop.additionalButton];assert(desktop.additional.hidden);
 [desktop action:desktop.advancedButton];assert(!desktop.advanced.hidden);assert(desktop.window==original);capture(@"advanced");[desktop action:desktop.advancedButton];assert(desktop.advanced.hidden);
 clipare_peers("Desktop-PC\nLaptop");clipare_status("Синхронизация включена",1,"● Desktop-PC\n○ Laptop");clipare_sync_status(0,"Синхронизация включена","");
 assert(desktop.emptyDevices.hidden);assert(!desktop.peerList.hidden);assert(desktop.peers.count==2);assert(desktop.syncState==0);
 for(NSString *appearance in @[NSAppearanceNameAqua,NSAppearanceNameDarkAqua]) {
  desktop.window.appearance=[NSAppearance appearanceNamed:appearance];[desktop.window.contentView setNeedsDisplay:YES];capture([appearance isEqualToString:NSAppearanceNameAqua]?@"home-light":@"home-dark");
  opaqueCards(desktop.views[@0]);capture([appearance isEqualToString:NSAppearanceNameAqua]?@"opaque-light":@"opaque-dark");
  [NSWorkspace.sharedWorkspace.notificationCenter postNotificationName:NSWorkspaceAccessibilityDisplayOptionsDidChangeNotification object:nil];
 }
 desktop.window.appearance=nil;
 fallbackCards(desktop.views[@0]);capture(@"material-fallback");
 clipare_sync_status(1,"Синхронизация приостановлена","");capture(@"paused");
 clipare_sync_status(2,"Синхронизация недоступна","Ожидание NetBird");assert(desktop.syncState==2);capture(@"degraded");
 clipare_discovered("Desktop-PC — 100.64.0.2\nLaptop — 100.64.0.3","Выберите устройство");assert(desktop.view==CPDiscovery);assert(desktop.found.count==2);assert(desktop.window==original);capture(@"discovery");
 [desktop back];assert(desktop.view==CPHome);
 clipare_discovered("","Автоматическое обнаружение недоступно");assert(desktop.foundList.hidden);assert(!desktop.connect.enabled);capture(@"discovery-empty");
 clipare_pair("Desktop-PC хочет подключиться","482 731",1);assert(desktop.view==CPPairing);assert(desktop.reject.tag==15);assert(!desktop.approve.hidden);assert([desktop.sas.stringValue isEqualToString:@"482 731"]);capture(@"pairing");
 clipare_alert("Сначала завершите подключение");[desktop back];assert(desktop.view==CPPairing);assert(desktop.pairIncoming);
 clipare_pair_close();assert(desktop.view==CPHome);assert(desktop.window==original);
 clipare_pair("Laptop","482 731",2);assert(desktop.approve.tag==18);clipare_pair_close();
 clipare_update_prompt("Установлена последняя версия Clipare\n\nТекущая версия: 0.6.0","","Понятно",0);assert(desktop.view==CPUpdate);assert(desktop.install.hidden);assert(!desktop.dismiss.hidden);capture(@"update-latest");
 clipare_update_prompt("Доступна новая версия","Обновить","Позже",20);assert(desktop.install.tag==20);capture(@"update-available");
 clipare_update_prompt("Загрузка обновления…","","",0);assert(desktop.install.hidden&&desktop.dismiss.hidden);[desktop back];assert(desktop.view==CPUpdate);capture(@"update-download");
 clipare_update_prompt("Не удалось обновить Clipare","Повторить","Закрыть",19);assert(desktop.install.tag==19);capture(@"update-error");
 clipare_update_close();assert(desktop.view==CPHome);
 clipare_alert("Настройки сохранены");assert(desktop.view==CPNotice);assert(desktop.window==original);[desktop back];assert(desktop.view==CPHome);
 NSUInteger topLevel=0;for(NSWindow *w in NSApp.windows)if(w==original||([w.title isEqualToString:@"Clipare"]&&w.visible))topLevel++;assert(topLevel==1);
 if(argc>1&&!strcmp(argv[1],"preview")){NSDate *until=[NSDate dateWithTimeIntervalSinceNow:120];while(until.timeIntervalSinceNow>0){int event=clipare_poll();if(event==4)break;if(event==11||event==12)clipare_discovered("Desktop-PC — 100.64.0.2","Выберите устройство");if(event==13)clipare_pair("Desktop-PC","482 731",1);if(event==14||event==15||event==16)clipare_pair_close();[NSThread sleepForTimeInterval:0.02];}}
 clipare_close();puts("PASS: single-window navigation, typed status, empty state, disclosures, pairing, updates");return 0;
} }
